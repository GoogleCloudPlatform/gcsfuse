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

data_dimension = 5
data_set_split_index = 15  # data at [0, data_set_split_index) belong to Master branch and [data_set_split_index, 2*data_set_split_index) belongs to PR branch.


for i in range(0, data_set_split_index, data_dimension):
  dataMaster = []
  dataMaster.append("Master")
  # Fetch Results for FileSize, Read, Write, RandRead, and RandWrite for Master
  for j in range(0, data_dimension):
    dataMaster.append(data[i + j])
  table.add_row(dataMaster)

  dataPR = []
  dataPR.append("PR")
  # Fetch Results for FileSize, Read, Write, RandRead, and RandWrite for PR
  for j in range(0, data_dimension):
    dataPR.append(data[i + j + 15])
  table.add_row(dataPR)

  dataNewline = []
  # Adding New line to differentiate FileSizes
  for j in range(0, data_dimension + 1):
    dataNewline.append("\n")
  table.add_row(dataNewline)

print(table)
