Building a gcsfuse release:

1.  Note the current commit that `master` points to. Call it `123abcd`.

2.  Use a viewer like [gitx](http://rowanj.github.io/gitx/) to examine the
    changes between the previous release and `123abcd`. Write up release notes.

3.  Choose a new version number according to the rules of [semantic
    versioning][semver]. Call it `v1.2.3`.

4.  Run `git tag -a v1.2.3`. Put the release notes in the tag, formatting
    according to the standard set by [previous tags][tags].

5.  Push the tag with `git push origin v1.2.3`.

6.  On a CentOS VM (where `rpm-build` is available), build a Linux release:

        mkdir -p ~/tmp/release
        go build github.com/googlecloudplatform/gcsfuse/tools/build_release
        ./build_release --version 1.2.3 --commit 123abcd --output_dir ~/tmp/release --rpm

7.  On an OS X machine, build an OS X release:

        mkdir -p ~/tmp/release
        go build github.com/googlecloudplatform/gcsfuse/tools/build_release
        ./build_release --version 1.2.3 --commit 123abcd --output_dir ~/tmp/release

8.  [Create a new release][new-release] on GitHub. Paste in the release notes
    and update the contents of `~/tmp/release` from the previous two steps.

9.  Find and replace in `docs/installing.md` to reference the new version
    number. For example: `%s/1\.2\.2/1.2.3/gc`

[semver]: http://semver.org/
[tags]: https://github.com/GoogleCloudPlatform/gcsfuse/tags
[new-release]: https://github.com/GoogleCloudPlatform/gcsfuse/releases/new
