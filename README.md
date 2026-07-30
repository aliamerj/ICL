# ICL: Infrastructure Configuration Language
**Status: early, experimental, work in progress.** Syntax and internals will change.
 
ICL is a small alternative syntax for infrastructure configuration that compiles
down to Terraform's [JSON Configuration Syntax](https://developer.hashicorp.com/terraform/language/syntax/json).
The generated `.tf.json` is read directly by the real `terraform`/`tofu` binary.

ICL doesn't reimplement the provider protocol, state management, plan/apply, or
anything else Terraform Core already does well. It's a compiler that targets a
stable, documented spec, not a competing runtime.
 
```
icl build main.ic   # -> main.tf.json
tofu init
tofu validate
```
 
## Why
 
HCL is good at what it does, but a few things about it are more ceremony than
necessity: four different ways to write a string, comma-or-newline ambiguity in
collections, provider names stuttered into every resource type
(`aws_instance`, `aws_vpc`, ...). ICL keeps everything about HCL/Terraform that
already works well — block syntax, `=` for attributes, the general shape of a
`resource`/`provider` block — and only changes the parts that were genuinely
confusing or verbose.
 
## Example
 
```hcl
# Configure the AWS provider used by all resources in this file.
provider aws {
  # Provider source from the registry.
  source = "hashicorp/aws"
  # Pin the provider version for reproducible deployments.
  version = "5.37.0"
  # AWS region where resources will be created.
  region = "eu-west-1"
}

# Create the main Virtual Private Cloud (VPC).
resource aws_vpc as demo_vpc {
  # Primary IPv4 network for the VPC.
  cidr_block = "10.0.0.0/16"
  # Resource tags help identify infrastructure.
  tags = {Name = "Terraform VPC"}
}

# Create a public subnet inside the VPC.
resource aws_subnet as public_subnet {
  # Reference another resource using its alias.
  vpc_id = demo_vpc.id
  # Subnet CIDR must be contained within the VPC CIDR.
  cidr_block = "10.0.0.0/24"
}

# Internet Gateway enables internet access for the VPC.
resource aws_internet_gateway as igw {
  # Attach the gateway to the VPC.
  vpc_id = demo_vpc.id
}

# Route table defines how network traffic is routed.
resource aws_route_table as public_rtb {
  # Associate the route table with the VPC.
  vpc_id = demo_vpc.id
}

# Default route sends all outbound traffic to the Internet Gateway.
resource aws_route as default_route {
  # Route table that receives this route.
  route_table_id = public_rtb.id
  # Match every IPv4 destination.
  destination_cidr_block = "0.0.0.0/0"
  # Forward traffic through the Internet Gateway.
  gateway_id = igw.id
}
```
 
`demo_vpc.id` compiles to `"${aws_vpc.demo_vpc.id}"` in the generated JSON —
resource references still defer to Terraform at apply time, exactly like real
HCL, since ICL genuinely can't know an AWS-assigned ID at compile time.
 
## What works today
 
- `provider` blocks — required `source`/`version`, optional `alias` (with
  correct array-vs-object JSON output when a provider has multiple aliased
  configurations), arbitrary nested config via string/int/float/bool/list/object
  literals
- `resource` blocks — named, addressable, with collision detection across the
  whole file
- Cross-resource references (`demo_vpc.id`) and provider selection
  (`provider = aws.east`)
- Full expression support: arithmetic, comparisons, parentheses, unary minus,
  with correct int/float promotion
- A real diagnostics system — every error carries a file/line/column, a message,
  and a hint, with parser error recovery so one typo doesn't hide the next five
- `icl build` (compile to `.tf.json`), `icl inspect` (debug view, pretty or
  JSON), `icl fmt` (canonical formatting)
- Basic `.ic` syntax highlighting for VS Code (`editors/vscode`)
Every feature above has been validated against a real `tofu validate` run, not
just internal tests.
 
## What doesn't exist yet
 
- `lookup` (data sources), `input` (variables), `output`, `let` (locals)
- Contextual keywords — `provider`/`resource`/`as` are reserved everywhere
  right now, not just at statement-start
- No LSP, no editor autocomplete/diagnostics beyond syntax highlighting
## Architecture
 
```
.ic source
  → lexer      (tokens)
  → parser     (AST : precedence-climbing expressions, block/attribute grammar)
  → eval       (resolves literals + references against a shared Registry)
  → tfjson     (compiles resolved config to Terraform JSON syntax)
  → main.tf.json
```
 
Each stage is its own package (`lexer`, `parser`, `eval`, `tfjson`,
`diagnostics`, `output`) rather than one monolith, so each is independently
testable and reusable. The AST uses one generic `Block` type for every keyword
(`provider`, `resource`, and future keywords) rather than a Go type per
keyword, adding a new keyword is a new parser `case` and a new `EvalX`
function, not a new AST hierarchy.
 
Built by hand in Go, no parser generators. Structure and general approach owe
a lot to *Crafting Interpreters* (Robert Nystrom).
 
## Building
 
```
go build -o icl ./cmd/icl
./icl build examples/main.ic
./icl inspect examples/main.ic
./icl fmt examples/main.ic
```
 
## Feedback
 
This is a genuine experiment, not a finished product. If you have thoughts on
the syntax, the compile-to-JSON approach, or whether this is worth continuing
at all — issues and comments very welcome.
 
