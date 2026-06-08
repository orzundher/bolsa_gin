import sys
import json

data = json.loads(sys.stdin.read())
print('Ultimas 5 entradas:')
for i in range(5, 0, -1):
    idx = -i
    print('  [{}]: {} -> {:.2f} EUR'.format(85 + idx, data["dates"][idx], data["utilities"][idx]))
