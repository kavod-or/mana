package web

const notFoundPage = `<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="robots" content="noindex">
  <title>404 · Mana</title>
  <link rel="icon" href="/static/favicon.png?v=2" type="image/png">
  <link rel="stylesheet" href="/static/styles.css?v=26">
</head>
<body class="landing-page">
  <main class="notice-card">
    <img src="/static/logo.png?v=2" alt="Mana" width="512" height="168">
    <p class="error-code">404</p>
    <h1>Hier ist noch nicht gedeckt.</h1>
    <p>Diese Seite gibt es nicht. Scanne den QR-Code vor Ort, um das richtige Menü zu öffnen.</p>
    <div class="notice-translation" lang="en">
      <h2>This table hasn’t been set.</h2>
      <p>This page couldn’t be found. Scan the QR code at the venue to open the right menu.</p>
    </div>
  </main>
  <footer class="notice-footer">
    <span class="app-version app-version-label">Powered by Mana v{{version}}</span>
    <a class="github-link" href="https://github.com/kavod-or/mana" aria-label="Mana on GitHub">GitHub ↗</a>
  </footer>
</body>
</html>`
