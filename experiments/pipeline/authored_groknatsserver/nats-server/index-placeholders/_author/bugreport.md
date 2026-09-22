Subject mapping destinations no longer parse.

A destination token `$1` used to mean "take source token 1". `{{wildcard(2)}}` used to mean the same thing for token 2. `{{partition(10,1,2,3)}}` used to mean "hash tokens 1, 2 and 3 into 10 partitions", and the same string with extra spaces and capital P used to work too.

After the last build those destination tokens are treated as ordinary literals. Streams and imports that remap `foo.*` onto `$1` or `{{wildcard(1)}}` just publish the placeholder text. Partitioned mappings all land on the same subject. No error is raised for the forms that used to work; the ones that used to be rejected also fail closed now.
