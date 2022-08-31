#!/bin/bash
## Impala Module and filesets
echo "Creating module[impala]..."
make create-module MODULE=impala
echo "Creating filesets for module[impala]"
impala_filesets=("impalad" "profile" "catalogd" "statestored")
for fileset in "${impala_filesets[@]}";do
  echo "Creating fileset[$fileset] in module[impala]"
  make create-fileset MODULE=impala FILESET="$fileset"
  echo "Created fileset[$fileset] in module[impala]"
done
echo "Created module[impala] with filesets."

