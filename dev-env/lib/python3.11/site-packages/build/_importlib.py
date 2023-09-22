import sys


if sys.version_info < (3, 8):
    import importlib_metadata as metadata
elif sys.version_info < (3, 9, 10) or (3, 10, 0) <= sys.version_info < (3, 10, 2):
    try:
        import importlib_metadata as metadata
    except ModuleNotFoundError:
        from importlib import metadata
else:
    from importlib import metadata

__all__ = ['metadata']
