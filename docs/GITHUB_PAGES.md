# GitHub Pages deployment

TagLock's documentation landing page is a dependency-free static site in
`site/`. It does not require Node.js, a package manager, a build service, or
third-party runtime assets.

## Repository setup

1. Open **Settings → Pages** in the GitHub repository.
2. Under **Build and deployment**, choose **GitHub Actions** as the source.
3. Push a change under `site/` to `main`, or run **Deploy documentation site**
   manually from the Actions tab.
4. Review the automatically created `github-pages` environment and restrict it
   to the default branch if stronger deployment protection is required.

The workflow publishes the `site/` directory with GitHub's official Pages
actions. It has only `contents: read`, `pages: write`, and `id-token: write`
permissions. Deployments are serialized through the `pages` concurrency group.

See GitHub's official documentation for
[custom Pages workflows](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages)
and
[publishing sources](https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site).

## Local preview

The files can be opened directly, but a local HTTP server more closely matches
Pages behavior. Use any trusted static server already installed on the system;
TagLock does not add a development-server dependency.

For example, when Python is available:

```sh
python -m http.server 8080 --directory site
```

Then open `http://localhost:8080`.

## Custom domains

Configure custom domains through GitHub repository settings. Do not add a
`CNAME` file until the domain owner has approved the exact hostname and DNS
records. A committed `CNAME` alone does not configure the GitHub-side domain.

## Updating the site

- Keep public claims synchronized with `README.md` and implemented behavior.
- Preserve keyboard navigation, focus states, reduced-motion behavior, and
  readable contrast.
- Do not add analytics, trackers, remote fonts, cookies, or account requirements
  without an explicit privacy review.
- Keep the deployment artifact free of source snapshots, credentials, and local
  development output.

