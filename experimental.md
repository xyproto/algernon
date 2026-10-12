## React, JSX and TypeScript

When a directory contains an `index.jsx` or `index.tsx` file, Algernon serves it as a complete React application. The JSX/TSX is bundled with esbuild, wrapped in an HTML page with the React runtime loaded, and served with the correct content type.

* If `style.css` or `style.gcss` is present in the same directory, it is used. Otherwise, a built-in default stylesheet is applied.
* The React version defaults to 19. A different major version (if it is available and built-in to Algernon) can be selected by placing a `// React: <N>` comment at the top of `index.jsx` or `index.tsx`:

        // React: 19

* Lua scripts in the same directory (e.g. `login.lua`, `data.lua`) can serve as API endpoints, making it possible to build full-stack applications with JSX for the frontend and Lua for the backend.

## WebAuthn

[WebAuthn/FIDO2](https://webauthn.io/) can be used for passwordless authentication. Credentials are stored in the database alongside regular user data. Four Lua/Teal functions are available:

* `WebAuthnBeginRegister(username)` — Begin a registration ceremony. Writes JSON options to the response. Returns `true` if successful. Passkeys can only be registered for an existing user, by that user while logged in.
* `WebAuthnFinishRegister(username)` — Finish registration. Reads the attestation response from the request body and stores the credential. Returns `true` if successful.
* `WebAuthnBeginLogin(username)` — Begin a login ceremony. Writes JSON options to the response. Returns `true` if successful.
* `WebAuthnFinishLogin(username)` — Finish login. Reads the assertion response from the request body. On success, also logs the user in via the standard session mechanism. Returns `true` if successful.

The relying party ID and origin are automatically derived from the request. Example Teal endpoints:

```lua
-- webauthn_register_begin.tl
local username: string = Username()
if not WebAuthnBeginRegister(username) then
  content("application/json")
  print(json({error = "Registration failed"}))
end
```

```lua
-- webauthn_register_finish.tl
local username: string = Username()
content("application/json")
print(json({ok = WebAuthnFinishRegister(username)}))
```
