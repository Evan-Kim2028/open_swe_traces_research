# Exported API — streamrenders

Reader{ContentType,ContentLength,Reader,Headers}, Data{ContentType,Data}, Redirect{Code,Request,Location}, String{Format,Data}; WriteString.

## Pre-existing callers

context.Data/Reader/String/Redirect handlers (incl. file-from-disk and stream paths).
