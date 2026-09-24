# API left after excision

```go
func indexPlaceHolders(token string) (int16, []int, int32, string, error)
```

Return tuple:

- `int16` transform kind (`NoTransform`, `BadTransform`, `Partition`, `Wildcard`, `SplitFromLeft`, `SplitFromRight`, `SliceFromLeft`, `SliceFromRight`, `Split`, `Left`, `Right`, `Random`)
- `[]int` source-token indexes used by the mapping function
- `int32` scalar argument (partition count, slice/split position, random ceiling)
- `string` string argument (split delimiter)
- `error` parse failure

Helpers still present in the same package: `getMappingFunctionArgs`, `transformIndexIntArgsHelper`, and the compiled `*MappingFunctionRegEx` matchers.
