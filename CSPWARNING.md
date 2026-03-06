# CSP Warning

Algernon sets strict HTTP headers by default. If these headers prevent your web application from working correctly, the browser will report violations back to the server and Algernon will log a warning.

## How it works

Algernon adds a `Content-Security-Policy-Report-Only` header alongside the default security headers. This instructs the browser to report policy violations (such as cross-origin framing) to a built-in endpoint (`/_csp_report`) without blocking anything beyond what the regular headers already enforce.

When the `--stricter` flag is used, the enforcing `Content-Security-Policy` header also includes `report-uri`, so any blocked request is reported as well.

Each unique violation is logged once, to avoid flooding the log.

## Testing the report-only warnings

These steps demonstrate how the `Content-Security-Policy-Report-Only` header reports framing violations from `X-Frame-Options: SAMEORIGIN`.

### 1. Start Algernon

Serve a directory with some HTML content:

```bash
mkdir -p /tmp/csptest
echo '<h1>Hello</h1>' > /tmp/csptest/index.html
algernon --httponly --debug --addr :4000 /tmp/csptest
```

### 2. Create a page on a different origin that frames the Algernon page

Start a second HTTP server on a different port:

```bash
mkdir -p /tmp/framer
cat > /tmp/framer/index.html << 'EOF'
<!doctype html>
<html>
<body>
  <h1>Framing test</h1>
  <iframe src="http://localhost:4000/" width="600" height="400"></iframe>
</body>
</html>
EOF
python3 -m http.server 4001 -d /tmp/framer
```

### 3. Open the framing page

Visit `http://localhost:4001/` in your browser and watch the Algernon log output. You should see a warning like:

```
WARN CSP report: "frame-ancestors 'self'" blocked "http://localhost:4001/" on http://localhost:4000/
```

The iframe will also be blocked by `X-Frame-Options: SAMEORIGIN`.

## Testing the enforcing warnings (--stricter)

With `--stricter`, the server sets a `Content-Security-Policy` that restricts `connect-src`, `object-src` and `form-action` to `'self'`. Violations are both blocked and reported.

### 1. Start Algernon with stricter headers

```bash
mkdir -p /tmp/csptest2
cat > /tmp/csptest2/index.html << 'EOF'
<!doctype html>
<html>
<body>
  <h1>Fetch test</h1>
  <script>
    fetch("https://httpbin.org/get")
      .then(r => r.text())
      .then(t => document.body.innerHTML += "<pre>" + t + "</pre>")
      .catch(e => document.body.innerHTML += "<p style='color:red'>" + e + "</p>");
  </script>
</body>
</html>
EOF
algernon --httponly --debug --stricter --addr :4000 /tmp/csptest2
```

### 2. Visit the page

Open `http://localhost:4000/` in your browser and check the Algernon log:

```
WARN CSP enforce: "connect-src 'self'" blocked "https://httpbin.org/get" on http://localhost:4000/
```

The fetch request is blocked by the CSP and the violation is reported back.

## Disabling security headers

If the headers are not appropriate for your application, you can disable them entirely:

```bash
algernon --noheaders /path/to/your/site
```

See `algernon --help` for all available flags.
