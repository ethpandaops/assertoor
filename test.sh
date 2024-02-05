#!/bin/bash

touch ./wiki.md

for filename in docs/*.md; do
    while IFS= read -r line; do
    if [[ "$line" =~ ^"#!!" ]]; then
        #echo "${line:3}"
        bash -c "cd . && ${line:3}" >> ./wiki.md
    else
        echo "$line" >> ./wiki.md
    fi
    done <<< $(cat $filename)
done
