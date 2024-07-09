# Copyright 2024 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# unit tests for custom_vm_perf_script

import unittest
import custom_vm_perf_script

class TestParseArguments(unittest.TestCase):
  def test_explicit_values(self):
    vm_name = "custom-vm"
    machine_type = "n2-standard-32"
    image_family = "debian-10"
    image_project = "debian-cloud"
    zone = "us-west1-a"
    startup_script = "custom_startup_script.sh"

    args = custom_vm_perf_script._parse_arguments(
        [
            "script",
            "--vm_name", vm_name,
            "--machine_type", machine_type,
            "--image_family", image_family,
            "--image_project", image_project,
            "--zone", zone,
            "--startup_script", startup_script
        ],
    )

    self.assertEqual(args.vm_name, vm_name)
    self.assertEqual(args.machine_type, machine_type)
    self.assertEqual(args.image_family, image_family)
    self.assertEqual(args.image_project, image_project)
    self.assertEqual(args.zone, zone)
    self.assertEqual(args.startup_script, startup_script)

  def test_default_values(self):
    args = custom_vm_perf_script._parse_arguments(["script"])

    self.assertEqual(args.vm_name, custom_vm_perf_script.DEFAULT_VM_NAME)
    self.assertEqual(args.machine_type, custom_vm_perf_script.DEFAULT_MACHINE_TYPE)
    self.assertEqual(args.image_family, custom_vm_perf_script.DEFAULT_IMAGE_FAMILY)
    self.assertEqual(args.image_project, custom_vm_perf_script.DEFAULT_IMAGE_PROJECT)
    self.assertEqual(args.zone, custom_vm_perf_script.DEFAULT_ZONE)
    self.assertEqual(args.startup_script, custom_vm_perf_script.DEFAULT_STARTUP_SCRIPT)


if __name__ == '__main__':
  unittest.main()
