# Commitments — httpresp

1. The error name is "HTTP response" plus " of <parent name>" when a
   parent is set. In-tree coverage: none. Inferable: partially.
2. Prepare installs empty mapped attributes for BOTH headers and cookies
   when they are nil. In-tree coverage: none. Inferable: partially.
3. A body is forbidden only for status codes 100-199, 204 and 304; every
   other status — including 205 and 3xx codes other than 304 — allows
   one. The check is skipped entirely for streaming endpoints, and only
   fires when the computed endpoint body is not empty. In-tree coverage:
   none. Inferable: no — the 205/304 set is RFC trivia.
4. A status of exactly 0 is "HTTP response status not defined".
   In-tree coverage: none. Inferable: partially.
5. On a JSON-RPC endpoint, a success response may not map result
   attributes to headers or cookies: non-empty mapped attributes on
   either produce dedicated errors. In-tree coverage: none. Inferable:
   doc — a comment explains shared batch/SSE responses.
6. ContentType "text/html" or "text/plain" (and not skipping body
   encode/decode) forces the result — or an explicit body — to be
   String or Bytes. In-tree coverage: none. Inferable: partially —
   doc-adjacent, the String-or-Bytes pair is the detail.
7. Header attribute names must resolve against the result type — for a
   result type the lookup is per-view and a missing name reports
   'attribute_name:header_name' notation with an "all views of" qualifier.
   Header values must be primitives or arrays of primitives; cookie
   values must be primitives ONLY (arrays rejected). In-tree coverage:
   deleted tests only. Inferable: no.
8. One header or cookie on a non-object result errors only when MORE
   than one is mapped; an array result mapped to a header must have
   primitive elements and to a cookie is always an error. In-tree
   coverage: deleted tests only. Inferable: no.
9. An explicit body with SkipResponseBodyEncodeDecode is an error; with
   no explicit body the computed body must be Empty under the same flag.
   Body attributes named by origin:attribute meta, or every field of an
   object body, must exist in the result type. In-tree coverage: none.
   Inferable: no.
10. Finalize wraps a non-empty object body in a generated user type
    named "<Service><Endpoint>ResponseBody", records the original type
    name under "name:original" meta, splits "name:header" tags at ":",
    and adds required body fields to the body validation. In-tree
    coverage: none. Inferable: no.
11. Finalize takes ContentType from the result type's own ContentType
    only when the response did not set one, and inherits headers/cookies
    from the service attribute. In-tree coverage: none. Inferable: no.
12. Dup copies scalar fields (including StatusCodeSet, Tag, Parent,
    Meta) and deep-copies body/headers/cookies. In-tree coverage: none.
    Inferable: partially.
13. For ErrorResult bodies only, attributes not mapped to body or
    headers are mapped to headers under "apikit-attribute-<name>" keys
    (already-mapped names skipped, requiredness propagated); a
    non-object error result maps under a single "apikit-attribute" key
    when headers and body are both empty. In-tree coverage: none.
    Inferable: no.
