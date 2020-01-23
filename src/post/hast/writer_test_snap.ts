// Jest Snapshot v1, https://goo.gl/fbAQLP

exports[`HastWriter formatting html > meta + meta + link + script 1`] = `
"<!doctype html>
<html lang=\\"en\\">
<head>
  <meta charset=\\"foo\\">
  <meta charset=\\"bar\\">
  <link rel=\\"icon\\" href=\\"/favicon.ico\\">
  <script defer src=\\"/baz.js\\" type=\\"module\\"></script>
</head>
<body>
  <header>
    <nav class=\\"site-nav\\" role=\\"navigation\\"><a class=\\"site-title\\" href=\\"/\\" title=\\"Home page\\">Joe Schafer</a>
      <ul>
        <li><a href=\\"https://github.com/jschaf\\" title=\\"GitHub page\\">GitHub</a></li>
        <li><a href=\\"https://www.linkedin.com/in/jschaf/\\" title=\\"LinkedIn page\\">LinkedIn</a></li>
      </ul>
    </nav>
  </header>
  <main>
    <div class=\\"main-inner-container\\">
      <p>foo</p>
    </div>
  </main>
  <footer role=\\"contentinfo\\"><a href=\\"/\\" title=\\"Home page\\">© 2020 Joe Schafer</a></footer>
</body>
</html>"
`;

exports[`HastWriter should compile body > p 1`] = `
"<!doctype html>
<html lang=\\"en\\">
<head>
  <meta charset=\\"utf-8\\">
  <meta name=\\"viewport\\" content=\\"width=device-width, initial-scale=1.0\\">
  <meta name=\\"robots\\" content=\\"index, follow\\">
  <link rel=\\"icon\\" href=\\"/favicon.ico\\">
  <link rel=\\"apple-touch-icon-precomposed\\" href=\\"/favicon-152.png\\">
  <link rel=\\"stylesheet\\" href=\\"/style/main.css\\">
  <script defer src=\\"/instantpage.min.js\\" type=\\"application/javascript\\"></script>
</head>
<body>
  <header>
    <nav class=\\"site-nav\\" role=\\"navigation\\"><a class=\\"site-title\\" href=\\"/\\" title=\\"Home page\\">Joe Schafer</a>
      <ul>
        <li><a href=\\"https://github.com/jschaf\\" title=\\"GitHub page\\">GitHub</a></li>
        <li><a href=\\"https://www.linkedin.com/in/jschaf/\\" title=\\"LinkedIn page\\">LinkedIn</a></li>
      </ul>
    </nav>
  </header>
  <main>
    <div class=\\"main-inner-container\\">
      <p>foo bar</p>
    </div>
  </main>
  <footer role=\\"contentinfo\\"><a href=\\"/\\" title=\\"Home page\\">© 2020 Joe Schafer</a></footer>
</body>
</html>"
`;
