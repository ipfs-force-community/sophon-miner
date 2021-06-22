package metrics

import (
	"context"
	"reflect"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/venus-miner/api"
)

func MetricedMinerAPI(a api.MinerAPI) api.MinerAPI {
	var out api.MinerStruct
	proxy(a, &out.Internal)
	proxy(a, &out.CommonStruct.Internal)
	return &out
}


func proxy(in interface{}, out interface{}) {
	rint := reflect.ValueOf(out).Elem()
	ra := reflect.ValueOf(in)

	for f := 0; f < rint.NumField(); f++ {
		field := rint.Type().Field(f)
		fn := ra.MethodByName(field.Name)

		rint.Field(f).Set(reflect.MakeFunc(field.Type, func(args []reflect.Value) (results []reflect.Value) {
			ctx := args[0].Interface().(context.Context)
			// upsert function name into context
			ctx, _ = tag.New(ctx, tag.Upsert(Endpoint, field.Name))
			stop := Timer(ctx, APIRequestDuration)
			defer stop()
			// pass tagged ctx back into function call
			args[0] = reflect.ValueOf(ctx)
			return fn.Call(args)
		}))

	}
}
