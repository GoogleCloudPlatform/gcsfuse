Building a gcsfuse release:

1.  Choose the commit at which you want to build a release. Save it to an
    environment variable:

        export COMMIT=123abcd

2.  Use a viewer like [gitx](http://rowanj.github.io/gitx/) to examine the
    changes between the previous release and `$COMMIT`. Write up release notes.

3.  Choose a new version number according to the rules of [semantic
    versioning][semver]. Save it to an environment variable:

        export VERSION=1.2.3

4.  Run `git tag -a v$VERSION $COMMIT`. Put the release notes in the tag,
    formatting according to the standard set by [previous tags][tags].

5.  Push the tag with `git push origin v$VERSION`.

6.  On a CentOS VM (where `rpm-build` is available), build Linux release
    packages:

        mkdir -p ~/tmp/release
        go get -v -u github.com/googlecloudplatform/gcsfuse/tools/package_gcsfuse
        package_gcsfuse ~/tmp/release $VERSION

7.  Sign the `.rpm` file generated in the previous step.

8.  [Create a new release][new-release] on GitHub. Paste in the release notes
    and upload the `.deb` and `.rpm` files built before.

9.  Find and replace in `docs/installing.md` to reference the new version
    number. For example: `%s/1\.2\.2/1.2.3/gc`

10. Update the Google Cloud packages server for both `apt-get` and `yum`.

11. Update the gcsfuse [formula][] in homebrew.

[semver]: http://semver.org/
[tags]: https://github.com/GoogleCloudPlatform/gcsfuse/tags
[new-release]: https://github.com/GoogleCloudPlatform/gcsfuse/releases/new
[formula]: https://github.com/Homebrew/homebrew-fuse/blob/master/Formula/gcsfuse.rb
