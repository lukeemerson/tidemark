import sys, html, re
for p in sys.argv[1:]:
    t = html.unescape(re.sub(r'<[^>]*>', '', open(p).read()))
    t = ''.join('.' if c == '⠀' else '#' if 0x2800 < ord(c) <= 0x28ff else c for c in t)
    print('=====', p)
    print('\n'.join(l.rstrip() for l in t.splitlines()))
