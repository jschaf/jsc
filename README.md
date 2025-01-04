# Code for Joe Schafer's blog

The code is tightly coupled to my own blog and not designed to be reusable in
any sense of the word. The design goals of the blog:

- Push config, compiling, and rendering into testable Go code.
- Keep markdown relatively simple. Move complexity into the parsing and rendering.
- Hot reload everything, code, CSS, markdown, JS.
- Avoid the node ecosystem. 

### Installation

```shell
sudo apt-get install libvips-dev liborc-0.4-dev libtiff5-dev \
     libjpeg-turbo8-dev libgsf-1-dev
```

### Dev server

The dev server has the following features:

- Hot reloads via the live reload protocol. The server injects the live-reload script into every HTML page.

- Compile markdown on change and refresh page.

- Reapply CSS changes on change without a full page refresh.

- Recompile the server when source code changes and replaces the 
  running server.

### Markdown extensions

Here's a list of markdown extensions I've created:

**Preview blocks**: I added preview blocks to add previews to links on hover. The
extension embeds the preview data into the link as data attributes like
`data-preview-title`. The main JavaScript file displays the embedded data on
hover.

```markdown
A [MOTD][motd-wiki] sends information to all users on login.

::: preview https://en.wikipedia.org/wiki/Motd_(Unix)
motd (Unix)

The **/etc/motd** is a file on Unix-like systems that contains a "message of the
day", used to send a common message to all users in a more efficient manner than
sending them all an e-mail message.
:::
```

**Small caps detection**: The markdown parser extracts text runs that look like
small caps.

- `TLA` - Converts any 3 consecutive uppercase ASCII letters to small caps.
- `(NASA)` - Converts wrapping parens as well as the letters to small caps.
- `TLAs` - Converts uppercase letters ending with a lower case s to small caps.

**Figures with captions**: Use `<figure>`, `<picture>` and `<figcaption>` for
figures:

```markdown
![alt text](./bar.png "title")

CAPTION: foobar
```

Converts to the following HTML:

```html
<figure>
    <picture>
        <img src="bar.png" alt="alt text" title="title">
    </picture>
    <figcaption>
        foobar
    </figcaption>
</figure>
```

**CONTINUE_READING**: When a line starts with `CONTINUE_READING`, the list view
of posts truncates the following content. For the detail view, the 
`CONTINUE_READING` is skipped.

**Citations (in progress)**: Citation support similar to Pandoc: https://pandoc.org/MANUAL.html#citations


Citations use square brackets with an @ sign followed by the bibtex identifier.
`[see @doe99, pp. 33-35]`
