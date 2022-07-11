import argparse
import configparser
from datetime import datetime as dt
import logging
import os
import subprocess
from subprocess import Popen
import sys
import ast


logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    handlers=[logging.StreamHandler(sys.stdout)],
)
log = logging.getLogger()

#def CompareDirectoryStructure(config):

def MountGCSBucket(bucketName):
  gcsBucket = "gcsBucket"
  subprocess.call("mkdir {}".format(gcsBucket), shell=True)
  subprocess.call("gcsfuse --implicit-dirs --disable-http2 --max-conns-per-host 100 {} {}".format(bucketName,gcsBucket), shell=True)
  return gcsBucket


def ParseArguments(argv):
    argv = sys.argv

    parser = argparse.ArgumentParser()

    parser.add_argument(
        "config_file", help="Provide path of the config file", action="store"
    )

    parser.add_argument(
        "--keep_files",
        help="Does not delete the directory structure in persistent disk.",
        action="store_true",
        default=False,
        required=False,
    )

    parser.add_argument(
        "--upload",
        help="Upload the results to the Google Sheet.",
        action="store_true",
        default=False,
        required=False,
    )

    parser.add_argument(
        "--message",
        help="Puts a message/title describing the test.",
        action="store",
        nargs=1,
        required=False,
    )

    args = parser.parse_args(argv[1:])

    return args


def ParseConfig(configFilePath):
    config = configparser.ConfigParser()
    config.read(os.path.abspath(configFilePath))

    configDict = {}

    configDict["bucketName"] = config["DEFAULT"]["bucket_name"]
    configDict["command"] = config["DEFAULT"]["command"]

    if "command_flags" in config["DEFAULT"]:
        configDict["commandFlags"] = ast.literal_eval(
            config.get("DEFAULT", "command_flags")
        )

    configDict["rootFolder"] = config["DEFAULT"]["root_folder"]

    count = 1

    for section in config.sections():
        folder = config[section]["folder"]
        numFiles = config[section]["num_of_files"]
        numSubDirectories = config[section]["num_of_subdir"]

        if int(config[section]["num_of_files"]) > 0:
            fileSize = config[section]["file_size"]

        if int(config[section]["num_of_files"]) > 0:
            fileNamePrefix = config[section]["file_name_prefix"]

        if int(config[section]["num_of_subdir"]) > 0:
            subDirectoryNamePrefix = config[section]["subdir_name_prefix"]

        subDirectoryDict = {
            "folder": folder,
            "numFiles": numFiles,
            "numSubDirectories": numSubDirectories,
            "fileSize": fileSize,
            "fileNamePrefix": fileNamePrefix,
            "subDirectoryNamePrefix": subDirectoryNamePrefix,
        }

        configDict["folder{}".format(count)] = subDirectoryDict

        count += 1

    configDict["numFolders"] = count - 1

    return configDict


def CheckDependencies(packages):
    for currPackage in packages:
        log.info("Checking whether {} is installed.".format(currPackage))

        exit_code = subprocess.call("dpkg -s {}".format(currPackage), shell=True)
        if exit_code != 0:
            log.error("{} not installed. Please install.".format(currPackage))
            subprocess.call("bash", shell=True)

    return


if __name__ == "__main__":

    args = ParseArguments(sys.argv)

    #  CheckDependencies(['gsutil', 'gcsfuse'])

    config = ParseConfig(args.config_file)

    gcsBucket = MountGCSBucket(config["bucketName"])

    #directoryStructurePresent = CompareDirectoryStructure(config)
