#!/usr/bin/env python3
"""Seeded coded-octet vectors. Semantics: TS 23.032 V19.0.0 §§6–8; polygon geometry excluded."""
import json,random,pathlib,sys
R=random.Random(230321900)
V=[]
def be(n,l):return (n%(1<<(8*l))).to_bytes(l,'big')
def add(name,b,valid=True,kind='shape',rule=None):
 V.append(dict(name=name,valid=valid,kind=kind,type=b[0]>>4,hex=b.hex(),**({'invalid_rule':rule} if rule else {})))
def shape(s,**kw):
 d=dict(lat=0,lon=0,n=3,alt=0,major=0,minor=0,orientation=0,confidence=0,ua=0,vc=0,radius=0,offset=0,included=0,hu=0,vu=0);d.update(kw)
 p=be(d['lat'],3 if s<11 else 4)+be(d['lon'],3 if s<11 else 4)
 b=bytes([s<<4])
 if s==5:return bytes([(s<<4)|d['n']])+p*d['n']
 b+=p
 if s in (8,9):b+=be(d['alt'],2)
 if s in (12,14):b+=be(d['alt']&0x3fffff,3)
 if s==1:b+=bytes([d['major']])
 if s in (3,9,11,12,13,14):
  b+=bytes([d['major'],d['minor'],d['orientation']])
  if s==9:b+=bytes([d['ua']])
  b+=bytes([d['confidence']|(d['hu']<<7)])
  if s in (12,14):b+=bytes([d['ua'],d['vc']|(d['vu']<<7)])
 if s==10:b+=be(d['radius'],2)+bytes([d['major'],d['offset'],d['included'],d['confidence']])
 return b
for s in (0,1,3,5,8,9,10,11,12,13,14):
 prefix=f's{s}'
 add(prefix+'_zero',shape(s))
 fields={'lat':([0x800000,0x7fffff,0xffffff] if s<11 else [-2147483648,-1,1073741824,2147483647]),'lon':([-8388608,-1,0,8388607] if s<11 else [-2147483648,-1,0,2147483647])}
 if s==5:fields['n']=[3,15]
 if s in (1,3,9,10,11,12,13,14):fields['major']=[0,1,2,20,127]+([253,254,255] if s>=11 else [])
 if s in (3,9,11,12,13,14):fields.update(minor=fields['major'],orientation=[0,90,179],confidence=[0,1,100,101,127])
 if s in (8,9):fields['alt']=[0,0x8000,0x7fff,0xffff]
 if s in (12,14):fields.update(alt=[-64000,-1,0,1280000],ua=fields['major'],vc=[0,1,100,101,127])
 if s==9:fields['ua']=[0,1,2,20,127]
 if s==10:fields.update(radius=[0,65535],offset=[0,179],included=[0,179],confidence=[0,1,100,101,127])
 for f,values in fields.items():
  for val in values:add(f'{prefix}_{f}_{val}',shape(s,**{f:val}))
 if s in (13,14):
  for hu in (0,1):
   for vu in ((0,1) if s==14 else (0,)):
    for k in (0,1,2,20,127,253,254,255):add(f'{prefix}_range_{hu}_{vu}_k{k}',shape(s,hu=hu,vu=vu,major=k,minor=k,ua=k))
 for i in range(20):
  d={f:R.choice(vals) for f,vals in fields.items()}
  d.update(lat=R.randrange(1<<24) if s<11 else R.randrange(-(1<<31),1<<31),lon=R.randrange(-(1<<23),1<<23) if s<11 else R.randrange(-(1<<31),1<<31))
  if s in (1,3,9,10,11,12,13,14):d['major']=R.randrange(256 if s>=11 else 128)
  if s in (3,9,11,12,13,14):d.update(minor=R.randrange(256 if s>=11 else 128),orientation=R.randrange(180),confidence=R.randrange(128))
  if s in (8,9):d['alt']=R.randrange(65536)
  if s in (12,14):d.update(alt=R.randrange(-64000,1280001),ua=R.randrange(256),vc=R.randrange(128))
  if s==9:d['ua']=R.randrange(128)
  if s==10:d.update(radius=R.randrange(65536),offset=R.randrange(180),included=R.randrange(180),confidence=R.randrange(128))
  if s==5:d['n']=R.choice([3,15])
  if s in (13,14):d.update(hu=R.randrange(2),vu=R.randrange(2) if s==14 else 0)
  b=shape(s,**d)
  if s==5:
   b=b[:1]+b''.join(be(R.randrange(1<<24),3)+be(R.randrange(-(1<<23),1<<23),3) for _ in range(d['n']))
  add(f'{prefix}_random_{i:02}',b)
def vel(t,bearing=0,speed=0,down=0,vertical=0,unc=0,vu=0):
 b=bytes([(t<<4)|(down<<1)|(bearing>>8),bearing&255])+be(speed,2)
 if t in (1,3):b+=bytes([vertical])
 if t in (2,3):b+=bytes([unc])
 if t==3:b+=bytes([vu])
 return b
for t in range(4):
 add(f'v{t}_zero',vel(t),kind='velocity')
 fields={'bearing':[0,359],'speed':[0,65535]}
 if t in (1,3):fields.update(down=[0,1],vertical=[0,255])
 if t in (2,3):fields['unc']=[0,255]
 if t==3:fields['vu']=[0,255]
 for f,vs in fields.items():
  for v in vs:add(f'v{t}_{f}_{v}',vel(t,**{f:v}),kind='velocity')
 for i in range(20):add(f'v{t}_random_{i:02}',vel(t,bearing=R.randrange(360),speed=R.randrange(65536),down=R.randrange(2) if t in (1,3) else 0,vertical=R.randrange(256),unc=R.randrange(256),vu=R.randrange(256)),kind='velocity')
def bad(name,b,rule,kind='shape'):add('invalid_'+name,b,False,kind,rule)
bad('header_spare',bytes([0x11])+shape(1)[1:],'§7.2 header spare bits')
bad('unc_spare',shape(1)[:-1]+b'\x80','§7.3.2 octet 8 bit 8 spare')
bad('reserved_shape',b'\x20'+b'\0'*6,'§7.2 reserved type 2')
bad('polygon_2',shape(5,n=2),'§7.3.4 count 3..15')
bad('orientation_180',shape(3,orientation=180),'§7.3.3 orientation 0..179')
bad('arc_offset_180',shape(10,offset=180),'§6.7 offset 0..179')
bad('arc_included_180',shape(10,included=180),'§6.7 included 0..179')
bad('bearing_360',vel(0,bearing=360),'§8.8 bearing 0..359','velocity')
bad('ha_alt_1280001',shape(12,alt=1280001),'§6.3a altitude -64000..1280000')
for kind,b in [('shape',shape(1)),('velocity',vel(3))]:
 bad(kind+'_length_minus',b[:-1],'exact length -1',kind);bad(kind+'_length_plus',b+b'\0','exact length +1',kind)
bad('velocity_spare',bytes([0x08])+vel(0)[1:],'§8.12 spare bits','velocity')
bad('velocity_reserved',bytes([0x40])+vel(0)[1:],'§8.6 reserved type 4','velocity')
pathlib.Path(sys.argv[1]).write_text(json.dumps(V,separators=(',',':'))+'\n')
print(f'Generated {len(V)} vectors')
