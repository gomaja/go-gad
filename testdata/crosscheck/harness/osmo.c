#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <osmocom/gsm/gad.h>
#include <osmocom/core/talloc.h>
static void jsonstr(const char *s) {
 putchar('"'); for (;*s;s++) {if (*s=='"'||*s=='\\') putchar('\\'); if ((unsigned char)*s<32) printf("\\u%04x",(unsigned char)*s); else putchar(*s);} putchar('"');
}
static unsigned load(const unsigned char *p, size_t n) {unsigned v=0;for(size_t i=0;i<n;i++)v=(v<<8)|p[i];return v;}
#define FIELD(x) printf(",\"" #x "\":%u",r->x)
#define ARR(x) printf(",\"" #x "\":%u",load(r->x,sizeof(r->x)))
#define POINT ARR(lat);ARR(lon)
static void raw_fields(const union gad_raw *raw) {
 printf("{\"type\":%u",raw->h.type);
 switch(raw->h.type) {
 case 0:{const struct gad_raw_ell_point *r=&raw->ell_point;POINT;break;}
 case 1:{const struct gad_raw_ell_point_unc_circle *r=&raw->ell_point_unc_circle;POINT;FIELD(unc);FIELD(spare2);break;}
 case 3:{const struct gad_raw_ell_point_unc_ellipse *r=&raw->ell_point_unc_ellipse;POINT;FIELD(unc_semi_major);FIELD(unc_semi_minor);FIELD(major_ori);FIELD(confidence);FIELD(spare1);FIELD(spare2);FIELD(spare3);break;}
 case 5:{printf(",\"num_points\":%u,\"points\":[",raw->polygon.h.num_points);for(unsigned i=0;i<raw->polygon.h.num_points;i++)printf("%s{\"lat\":%u,\"lon\":%u}",i?",":"",load(raw->polygon.point[i].lat,3),load(raw->polygon.point[i].lon,3));printf("]");break;}
 case 8:{const struct gad_raw_ell_point_alt *r=&raw->ell_point_alt;POINT;ARR(alt);break;}
 case 9:{const struct gad_raw_ell_point_alt_unc_ell *r=&raw->ell_point_alt_unc_ell;POINT;ARR(alt);FIELD(unc_semi_major);FIELD(unc_semi_minor);FIELD(major_ori);FIELD(unc_alt);FIELD(confidence);FIELD(spare1);FIELD(spare2);FIELD(spare3);FIELD(spare4);break;}
 case 10:{const struct gad_raw_ell_arc *r=&raw->ell_arc;POINT;ARR(inner_r);FIELD(unc_r);FIELD(ofs_angle);FIELD(incl_angle);FIELD(confidence);FIELD(spare1);FIELD(spare2);break;}
 case 11:{const struct gad_raw_ha_ell_point_unc_ell *r=&raw->ha_ell_point_unc_ell;POINT;ARR(alt);FIELD(unc_semi_major);FIELD(unc_semi_minor);FIELD(major_ori);FIELD(confidence);FIELD(spare1);break;}
 case 12:{const struct gad_raw_ha_ell_point_alt_unc_ell *r=&raw->ha_ell_point_alt_unc_ell;POINT;ARR(alt);FIELD(unc_semi_major);FIELD(unc_semi_minor);FIELD(major_ori);FIELD(h_confidence);FIELD(unc_alt);FIELD(v_confidence);FIELD(spare1);FIELD(spare2);break;}
 }
 printf("}");
}
int main(void) {
 char line[1024];
 while(fgets(line,sizeof(line),stdin)) {
  unsigned char b[256]={0}; size_t n=strcspn(line,"\r\n")/2;
  if(n>255) return 2;
  for(size_t i=0;i<n;i++) {unsigned x; if(sscanf(line+2*i,"%2x",&x)!=1) return 2; b[i]=(unsigned char)x;}
  union gad_raw raw={0}; struct osmo_gad gad={0}; struct osmo_gad_err *err=NULL;
  int rc=osmo_gad_raw_read(&raw,&err,NULL,b,n);
  printf("{\"raw_read_rc\":%d",rc);
  if(rc) {printf(",\"raw_error\":");jsonstr(err->logmsg);talloc_free(err);puts("}");continue;}
  printf(",\"raw\":{\"type\":%u,\"header_low_nibble\":%u,\"octets\":[",raw.h.type,b[0]&15);
  for(size_t i=0;i<n;i++)printf("%s%u",i?",":"",((unsigned char*)&raw)[i]);
  if(raw.h.type==1)printf("],\"lat\":%u,\"lon\":%u,\"unc\":%u,\"spare2\":%u}",(b[1]<<16)|(b[2]<<8)|b[3],(b[4]<<16)|(b[5]<<8)|b[6],raw.ell_point_unc_circle.unc,raw.ell_point_unc_circle.spare2);
  else printf("]}");
  printf(",\"raw_fields\":");raw_fields(&raw);
  rc=osmo_gad_dec(&gad,&err,NULL,&raw);printf(",\"decode_rc\":%d",rc);
  if(rc) {printf(",\"decode_error\":");jsonstr(err->logmsg);talloc_free(err);}
  else {char s[1024];int r=osmo_gad_to_str_buf(s,sizeof(s),&gad);printf(",\"decoded\":{\"type\":%d,\"lat_microdegrees\":%d,\"lon_microdegrees\":%d,\"unc_mm\":%u},\"to_string_rc\":%d,\"to_string\":",gad.type,gad.ell_point_unc_circle.lat,gad.ell_point_unc_circle.lon,gad.ell_point_unc_circle.unc,r);jsonstr(s);}
  puts("}");
 }
 return 0;
}
