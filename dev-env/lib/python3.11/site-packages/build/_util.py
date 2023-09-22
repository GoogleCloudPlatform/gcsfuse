from __future__ import annotations

import re
import sys
import tarfile
import typing

from collections.abc import Iterator, Set


_WHEEL_FILENAME_REGEX = re.compile(
    r'(?P<distribution>.+)-(?P<version>.+)'
    r'(-(?P<build_tag>.+))?-(?P<python_tag>.+)'
    r'-(?P<abi_tag>.+)-(?P<platform_tag>.+)\.whl'
)


def check_dependency(
    req_string: str, ancestral_req_strings: tuple[str, ...] = (), parent_extras: Set[str] = frozenset()
) -> Iterator[tuple[str, ...]]:
    """
    Verify that a dependency and all of its dependencies are met.

    :param req_string: Requirement string
    :param parent_extras: Extras (eg. "test" in myproject[test])
    :yields: Unmet dependencies
    """
    import packaging.requirements

    from ._importlib import metadata

    req = packaging.requirements.Requirement(req_string)
    normalised_req_string = str(req)

    # ``Requirement`` doesn't implement ``__eq__`` so we cannot compare reqs for
    # equality directly but the string representation is stable.
    if normalised_req_string in ancestral_req_strings:
        # cyclical dependency, already checked.
        return

    if req.marker:
        extras = frozenset(('',)).union(parent_extras)
        # a requirement can have multiple extras but ``evaluate`` can
        # only check one at a time.
        if all(not req.marker.evaluate(environment={'extra': e}) for e in extras):
            # if the marker conditions are not met, we pretend that the
            # dependency is satisfied.
            return

    try:
        dist = metadata.distribution(req.name)
    except metadata.PackageNotFoundError:
        # dependency is not installed in the environment.
        yield (*ancestral_req_strings, normalised_req_string)
    else:
        if req.specifier and not req.specifier.contains(dist.version, prereleases=True):
            # the installed version is incompatible.
            yield (*ancestral_req_strings, normalised_req_string)
        elif dist.requires:
            for other_req_string in dist.requires:
                # yields transitive dependencies that are not satisfied.
                yield from check_dependency(other_req_string, (*ancestral_req_strings, normalised_req_string), req.extras)


def parse_wheel_filename(filename: str) -> re.Match[str] | None:
    return _WHEEL_FILENAME_REGEX.match(filename)


if typing.TYPE_CHECKING:
    TarFile = tarfile.TarFile

else:
    # Per https://peps.python.org/pep-0706/, the "data" filter will become
    # the default in Python 3.14. The first series of releases with the filter
    # had a broken filter that could not process symlinks correctly.
    if (
        (3, 8, 18) <= sys.version_info < (3, 9)
        or (3, 9, 18) <= sys.version_info < (3, 10)
        or (3, 10, 13) <= sys.version_info < (3, 11)
        or (3, 11, 5) <= sys.version_info < (3, 12)
        or (3, 12) <= sys.version_info < (3, 14)
    ):

        class TarFile(tarfile.TarFile):
            extraction_filter = staticmethod(tarfile.data_filter)

    else:
        TarFile = tarfile.TarFile
