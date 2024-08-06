import subprocess

def check_dependencies(packages,log) -> None:
  """Check whether the dependencies are installed or not.

  Args:
    packages: List containing the names of the dependencies to be checked.

  Raises:
    Aborts the execution if a particular dependency is not found.
  """

  for curr_package in packages:
    log.info('Checking whether %s is installed.\n', curr_package)
    exit_code = subprocess.call(
        '{} --version'.format(curr_package), shell=True)
    if exit_code != 0:
      log.error(
          '%s not installed. Please install. Aborting!\n', curr_package)
      subprocess.call('bash', shell=True)

  return