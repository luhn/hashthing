# hashthing

*hashthing* is a utility for adding a hash of the file contents to the filename.
This solves two problems for web sites/applications:

1) Delivering the correct assets — 	When you deploy a new version of your application, pages from the old version may still be in-flight and will need to load old assets.
  This issue is exacerbated by rolling deployments, canaries, rollbacks, and other features of modern infrastructure.
  Adding a hash to the filename ensures that clients will always receive the appropriate assets.
2) Cache busting — The fewer assets a webpage needs to load, the faster your application will be available to the user.
  With a hash in the filename and an aggressive cache policy on your static assets,
  a client will load an asset once and won't load it again as long as the contents don't change.
  As soon as the contents do change, the hash changes and the new file is fetched.

## How to use it

Download the latest binary from the [releases page](https://github.com/luhn/hashthing/releases).
The binary is standalone and requires no external dependencies.
You can run directly from the local directory (e.g. `./hashthing`) or copy the file to a somewhere in your `PATH` (e.g. `mv hashthing /usr/bin/`).

Hashthing is a command line utility and requires two arguments, `[src]` and `[dst]`.
`[src]` is the directory from which assets will be sourced, `[dst]` is the directory that files will be copied to.
A mapping of source file paths to destination paths will be outputted to `manifest.json` in the local directory.
You can use the `manifest` flag to change this, e.g. `hashthing -manifest=app/rev-manifest.json src dst`.

A Docker image is also available as [luhn/hashthing](https://hub.docker.com/r/luhn/hashthing).
You can run the Docker image with the same arguments as the CLI:
`docker run luhn/hashthing src dst`

## How it works

Hashthing recursively walks through the source directory and for each source file performs the following actions:

1) If a CSS file, the file is scanned for `url()` references and the replaces the path with the appropriate destination path.
2) The file contents are hashed.
3) The file is copied into the destination directory with the hash inserted into the filename.

A mapping of source filenames to destinations filenames is outputted as JSON into the current directory under `manifest.json`.
The file might look something like this:

```json
{
  "main.css": "main.a5eeef51.css",
  "images/test.jpg": "images/test.55bc5514.jpg"
}
```

## Integrating with your application

To make use of hasthing, you'll need to update your application to load `manifest.json` and the parse the JSON object.
Asset paths should be translated using the manifest mapping.
How to best do this depends on your application.

Output assets can be copied to an S3 bucket or any other static hosting solution.
Oftentimes it's desirable to put a CDN in front of your static host.
Put sure to configure the host with an aggressive cache policy.
For example, to copy your assets to an S3 bucket with a one year cache policy, you can use the following command:

```bash
aws s3 sync --size-only --cache-control max-age=63072000 dst s3://mysite-static/
```

You may wonder how to clean up old assets from your storage;
our recommendation is to store old assets indefinitely.
Storage is so cheap that the few pennies saved by expiring old assets is not worth the effort of solving a surprisingly difficult problem.

## Prior art

This is not a new concept:

* Webpack has [output filenames](https://webpack.js.org/guides/caching/#output-filenames)
* Ruby on Rails has [sprockets](https://guides.rubyonrails.org/asset_pipeline.html#main-features)
* Django has [ManifestStaticFilesStorage](https://docs.djangoproject.com/en/3.0/ref/contrib/staticfiles/#django.contrib.staticfiles.storage.ManifestStaticFilesStorage)

These are opinionated, well-integrated features of larger frameworks.
hashthing operates independent of any framework, so can be used in tandem with frameworks that may not provide this feature or bespoke frameworks.
