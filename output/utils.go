package output

import (
	"fmt"
	"sort"

	"github.com/aliamerj/icl/eval"
)

func formatValue(v eval.Value) string {
	switch v.Kind {
	case eval.KindString:
		return fmt.Sprintf("%q", v.Str)
	case eval.KindInt:
		return fmt.Sprintf("%d", v.Int)
	case eval.KindFloat:
		return fmt.Sprintf("%g", v.Float)
	case eval.KindBool:
		return fmt.Sprintf("%t", v.Bool)
	case eval.KindNull:
		return "null"
	default:
		return "?"
	}
}

func sortedKeys(m map[string]eval.Value) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// toNative converts an eval.Value to the one Go type that actually
// represents it — this is the single place that "unwraps" the tagged
// union, so JSON, and any future output format, stays in sync by
// only changing this one function.
func toNative(v eval.Value) any {
	switch v.Kind {
	case eval.KindString:
		return v.Str
	case eval.KindInt:
		return v.Int
	case eval.KindFloat:
		return v.Float
	case eval.KindBool:
		return v.Bool
	case eval.KindNull:
		return nil
	default:
		return nil
	}
}
