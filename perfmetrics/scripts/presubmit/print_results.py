# Copyright 2023 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from prettytable import PrettyTable

data = []
for line in open("result.txt", "r").readlines():
  data.append(line.strip())

table = PrettyTable(
    ["Branch", "File Size", "Read BW", "Write BW", "RandRead BW", "RandWrite BW"]
)

DATA_DIMENSION  = 5
DATA_SET_SPLIT_INDEX  = 15  # data at [0, DATA_SET_SPLIT_INDEX ) belong to Master branch and [DATA_SET_SPLIT_INDEX , TOTAL_SIZE = 2 * DATA_SET_SPLIT_INDEX ) belongs to PR branch.


for i in range(0, DATA_SET_SPLIT_INDEX , DATA_DIMENSION ):
  dataMaster = []
  dataMaster.append("Master")
  # Fetch Results for FileSize, Read, Write, RandRead, and RandWrite for Master
  for j in range(0, DATA_DIMENSION ):
    dataMaster.append(data[i + j])
  table.add_row(dataMaster)

  dataPR = []
  dataPR.append("PR")
  # Fetch Results for FileSize, Read, Write, RandRead, and RandWrite for PR
  for j in range(0, DATA_DIMENSION ):
    dataPR.append(data[i + j + 15])
  table.add_row(dataPR)

  dataNewline = []
  # Adding New line to differentiate FileSizes
  for j in range(0, DATA_DIMENSION  + 1):
    dataNewline.append("\n")
  table.add_row(dataNewline)

print(table)
