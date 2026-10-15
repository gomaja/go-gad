#!/usr/bin/env python3
"""Wrap GAD in Diameter AVP 1242 and velocity in LwM2M Location resource 6/4."""
import json,struct,pathlib,sys
V=json.loads(pathlib.Path(sys.argv[1]).read_text())
def u24(x):return x.to_bytes(3,'big')
def checksum(b):
 if len(b)%2:b+=b'\0'
 s=sum(struct.unpack('!%dH'%(len(b)//2),b));s=(s&65535)+(s>>16);s=(s&65535)+(s>>16);return (~s)&65535
packets=[]
for i,v in enumerate(V):
 b=bytes.fromhex(v['hex'])
 if v['kind']=='shape':
  avp=struct.pack('!I',1242)+b'\xc0'+u24(12+len(b))+struct.pack('!I',10415)+b
  avp+=b'\0'*((-len(avp))%4)
  payload=b'\x01'+u24(20+len(avp))+b'\xc0'+u24(8388620)+struct.pack('!III',16777255,i+1,i+1)+avp
  seg=struct.pack('!HHIIBBHHH',20000+i,3868,1,1,0x50,0x18,65535,0,0)+payload;proto=6
 else:
  tlv=(bytes([0xc0|len(b),4]) if len(b)<8 else bytes([0xc8,4,len(b)]))+b
  payload=b'\x40\x03'+struct.pack('!H',i+1)+b'\xb1' +b'6'+ b'\x01'+b'0'+b'\x12\x2d\x16\xff'+tlv
  seg=struct.pack('!HHHH',20000+i,5683,8+len(payload),0)+payload;proto=17
 src=b'\xc0\x00\x02\x01';dst=b'\xc0\x00\x02\x02'
 if proto==6:
  c=checksum(src+dst+bytes([0,proto])+struct.pack('!H',len(seg))+seg);seg=seg[:16]+struct.pack('!H',c)+seg[18:]
 ip=struct.pack('!BBHHHBBH4s4s',0x45,0,20+len(seg),i,0,64,proto,0,src,dst)
 ip=ip[:10]+struct.pack('!H',checksum(ip))+ip[12:]
 packets.append(b'\x02\0\0\0\0\x02\x02\0\0\0\0\x01\x08\x00'+ip+seg)
with pathlib.Path(sys.argv[2]).open('wb') as f:
 f.write(struct.pack('<IHHIIII',0xa1b2c3d4,2,4,0,0,65535,1))
 for i,p in enumerate(packets):f.write(struct.pack('<IIII',1792022400+i,0,len(p),len(p))+p)
