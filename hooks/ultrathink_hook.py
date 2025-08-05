#!/usr/bin/env python3
import json
import sys

input_data = json.load(sys.stdin)
prompt = input_data.get("prompt", "")

if prompt.rstrip().endswith("-u"):
    modified_prompt = prompt.rstrip()[:-2].rstrip()
    
    context = f"{modified_prompt}\n\nultrathink"
    
    print(context)
else:
    print(prompt)

sys.exit(0)