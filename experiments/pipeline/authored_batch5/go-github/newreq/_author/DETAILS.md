# DETAILS — newreq

1. Every request built by NewRequest carries an `X-Github-Api-Version`
   header whose value is the client's configured default API version.
   Inferable: doc — GitHub documents the version header, and the client
   field `apiVersionDefault` exists; the specific literal is arbitrary.
2. When the client's user-agent string is empty, the `User-Agent` header
   is omitted from the request entirely rather than sent with an empty
   value. Inferable: partially — omitting empty headers is conventional,
   but nothing forces it; a solver could equally send `User-Agent: ""`.
3. When user-agent is non-empty it is sent verbatim. Inferable: yes.
