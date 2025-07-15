package ntnn

import (
	"context"
	"fmt"
	"runtime/pprof"
)

func WithLabels(
	ctx context.Context,
	fn func(context.Context),
	labelPairs ...any,
) {
	if ctx == nil {
		ctx = context.Background()
	}

	strLabelPairs := make([]string, len(labelPairs))
	for i := range labelPairs {
		strLabelPairs[i] = fmt.Sprintf("%v", labelPairs[i])
	}

	pprof.Do(
		ctx,
		pprof.Labels(strLabelPairs...),
		fn,
	)
}
