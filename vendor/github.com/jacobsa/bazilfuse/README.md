This package is a fork of [bazil.org/fuse][upstream], used as an implementation
detail of [github.com/jacobsa/fuse][fuse]. If you are looking at this, the
latter package is probably the one you want to use, rather than using this
directly.

[fuse]: https://github.com/jacobsa/fuse
[upstream]: https://github.com/bazillion/fuse

Changes from upstream:

*   The function SetOption allows for setting arbitrary mount options (see
    see [issue 77](https://github.com/bazillion/fuse/issues/77)).
