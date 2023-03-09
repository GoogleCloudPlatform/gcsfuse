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

