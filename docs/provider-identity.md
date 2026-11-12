# Provider identity and safe recovery (PR #16)

In registry mode, each job now persists a provider identity before submission. The identity is a SHA-256 fingerprint of adapter kind, configured provider name and endpoint, not a secret or authentication token. Recovery checks the persisted identity against the current model route before issuing any upstream GET. A mismatch fails the job closed, without contacting the newly routed provider or resubmitting a video request.

Existing in-flight jobs created before this version have no provider identity. In registry mode, those jobs fail closed after restart; they cannot be safely associated with an upstream provider. Finished historical jobs remain readable.

Do not change provider endpoints or model mappings while remote jobs are active unless you accept failed recovery. Provider identity is not a cryptographic authentication of the remote service; infrastructure ownership, network routing, and credentials must be protected separately. Credentials are intentionally excluded from the fingerprint to permit safe token rotation. For programmatic registry construction, supply an explicit stable `Identity`; when omitted it defaults to the entry name for backwards compatibility.
