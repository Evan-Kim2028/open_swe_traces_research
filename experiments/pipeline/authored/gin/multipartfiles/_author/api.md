# Exported API — multipartfiles

reached through FormMultipart binding; exported error values ErrMultiFileHeader and ErrMultiFileHeaderLenInvalid.

## Pre-existing callers

formMultipartBinding.Bind via mappingByPtr(obj, (*multipartRequest)(req), "form")
