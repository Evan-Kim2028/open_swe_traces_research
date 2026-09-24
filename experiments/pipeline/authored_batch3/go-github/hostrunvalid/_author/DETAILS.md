# DETAILS — hostrunvalid

1. An empty Name is rejected. Inferable: doc — the doc comment says required
   fields are validated, and the request type marks Name required.
2. A zero-value Image is rejected. Inferable: doc.
3. An empty Size is rejected. Inferable: doc.
4. A zero RunnerGroupID is rejected. Inferable: doc.
5. When all four fields are present the request is accepted and the call
   proceeds to the API. Inferable: yes.
6. Each rejection produces a field-specific error message. Inferable: no —
   the literals are arbitrary; verifiers should check only that an error
   surfaces.
