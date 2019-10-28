#!/bin/bash

# Created
cat results/* | awk '{print substr($6, 0, 7), $4, $3}' \
  | grep stephen-soltesz | sort | uniq | awk '{print $1, $2}' | sort | uniq -c

cat results/* | awk '{if ($6 > "2017-01-01T00:00:00Z") { print $0}}' \
  | awk '{print $3, $2, $4, substr($6, 0, 10)}' \
  | sort -n | uniq | awk '{print $3}' | sort | uniq -c | sort -n > created.txt

# Reviewed
cat results/* | awk '{print substr($6, 0, 7), $5, $3}' \
  | grep stephen-soltesz | sort | uniq | awk '{print $1, $2}' | sort | uniq -c

cat results/* | awk '{if ($6 > "2017-01-01T00:00:00Z") { print $0}}' \
  | awk '{print $3, $2, $5, substr($6, 0, 10)}' \
  | sort -n | uniq | awk '{print $3}' | sort | uniq -c | sort -n > reviewed.txt
