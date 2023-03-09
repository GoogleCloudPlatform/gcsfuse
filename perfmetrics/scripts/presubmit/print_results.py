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

Names = []
for line in open('result.txt','r').readlines():
  Names.append(line.strip())

t = PrettyTable(["Branch",'File Size', "Read BW", "Write BW", "RandRead BW", "RandWrite BW"])

for i in range(0,15,5) :
  dataMaster = []
  dataMaster.append("Master")
  for j in range(0,5) :
    dataMaster.append(Names[i+j])

  t.add_row(dataMaster)

  dataPR = []
  dataPR.append("PR")
  for j in range(0,5) :
    dataPR.append(Names[i+j+15])

  t.add_row(dataPR)

print(t)