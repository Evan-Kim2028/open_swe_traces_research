# Bug report

Terraform expression construction is broken: resource references, data references, lists, function calls and conditionals render with wrong or missing text — names keep illegal characters, strings lose their quoting, list brackets and argument separators are wrong, and output variables come out unsorted with duplicates.

Expected: a resource property renders like `aws_instance.web.id`; a data source renders like `data.aws_ami.ubuntu.id`; a list renders `[a, b]`; a string literal renders `"value"`; an int renders `3`; a conditional over an empty-string check renders `expr == "" ? null : value`.

Reproduce with:

```
go test -count=1 ./upup/pkg/fi/cloudup/terraformWriter/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
