import{y as B,_ as D}from"./js-yaml-ocLjxu-C.js";import{e as C}from"./traffic-XuFzmEwh.js";import{d as R,u as b,n as u,Q as I,v as r,t as e,e as c,o as s,c as a,z as p,B as L,h as x,a6 as N,a7 as S,J as f,f as A,y as M,F as O,K as T,G as Y,H as $,_ as F}from"./index-eYv_Far2.js";import"./request-WeLgEOdi.js";const h=n=>(Y("data-v-4f1417af"),n=n(),$(),n),j={class:"editorBox"},q=h(()=>f("p",null,"修改时间: 2024/3/20 15:20:31",-1)),z=h(()=>f("p",null,"版本号: xo842xqpx834",-1)),E=R({__name:"YAMLView",setup(n){const k=b(),V=u(!0),t=u(!1),m=u(8),y=u(`configVersion: v3.0
force: true
enabled: true
key: shop-detail
tags:
  - name: gray
    match:
      - key: env
        value:
          exact: gray`),w=async()=>{var l;const o=await C((l=k.params)==null?void 0:l.ruleName);o.code===200&&(y.value=B.dump(o==null?void 0:o.data))};return I(()=>{w()}),(o,l)=>{const d=c("a-button"),_=c("a-flex"),v=c("a-col"),i=c("a-card");return s(),r(i,null,{default:e(()=>[a(_,{style:{width:"100%"}},{default:e(()=>[a(v,{span:t.value?24-m.value:24,class:"left"},{default:e(()=>[a(_,{vertical:"",align:"end"},{default:e(()=>[a(d,{type:"text",style:{color:"#0a90d5"},onClick:l[0]||(l[0]=g=>t.value=!t.value)},{default:e(()=>[p(L(o.$t("flowControlDomain.versionRecords"))+" ",1),t.value?(s(),r(x(S),{key:1})):(s(),r(x(N),{key:0}))]),_:1}),f("div",j,[a(D,{modelValue:y.value,theme:"vs-dark",height:500,language:"yaml",readonly:V.value},null,8,["modelValue","readonly"])])]),_:1})]),_:1},8,["span"]),a(v,{span:t.value?m.value:0,class:"right"},{default:e(()=>[t.value?(s(),r(i,{key:0,class:"sliderBox"},{default:e(()=>[(s(),A(O,null,M(2,g=>a(i,{key:g},{default:e(()=>[q,z,a(_,{justify:"flex-end"},{default:e(()=>[a(d,{type:"text",style:{color:"#0a90d5"}},{default:e(()=>[p("查看")]),_:1}),a(d,{type:"text",style:{color:"#0a90d5"}},{default:e(()=>[p("回滚")]),_:1})]),_:1})]),_:2},1024)),64))]),_:1})):T("",!0)]),_:1},8,["span"])]),_:1})]),_:1})}}}),P=F(E,[["__scopeId","data-v-4f1417af"]]);export{P as default};
