import{u as F}from"./index-Haky1Klh.js";import{d as G,g as L,u as A,n as h,r as H,a as J,O as U,f as v,c as t,t as e,e as l,o as i,y as a,z as n,v as m,h as w,a8 as K,a9 as Q,H as C,V as B,F as S,x as O,J as X,W as Y,E as Z,G as ee,_ as te}from"./index-0c4Jk4Xf.js";import{g as oe}from"./traffic-hA30ZvQx.js";import"./request-fr6f-AQz.js";const M=b=>(Z("data-v-b91b849f"),b=b(),ee(),b),ae={class:"__container_routingRule_detail"},le=M(()=>C("p",null,"修改时间: 2024/3/20 15:20:31",-1)),ne=M(()=>C("p",null,"版本号: xo842xqpx834",-1)),se=G({__name:"formView",setup(b){const{appContext:{config:{globalProperties:P}}}=L(),E=A(),d=h(!1),R=h(8),q=F().toClipboard;function V(o){Y.success(P.$t("messageDomain.success.copy")),q(o)}const s=H({configVersion:"v3.0",scope:"service",key:"org.apache.dubbo.samples.UserService",enabled:!0,runtime:!0,force:!1,conditions:["=>host!=192.168.0.68"]}),I=J(()=>{const o=s.key.split(":");return o[0]?o[0]:""}),D=h([]),W=h([]);async function x(){var r;let o=await oe((r=E.params)==null?void 0:r.ruleName);console.log(o),(o==null?void 0:o.code)===200&&(Object.assign(s,(o==null?void 0:o.data)||{}),s.conditions.forEach(_=>{const c=_.split(" & "),p=c[1].split(" => ");D.value.push(c[0]),D.value.push(p[0]),W.value.push(p[1])}))}return U(()=>{x()}),(o,r)=>{const _=l("a-typography-title"),c=l("a-button"),p=l("a-flex"),f=l("a-descriptions-item"),y=l("a-typography-paragraph"),z=l("a-descriptions"),g=l("a-card"),T=l("a-row"),N=l("a-tag"),$=l("a-space"),j=l("a-col");return i(),v("div",ae,[t(p,{style:{width:"100%"}},{default:e(()=>[t(j,{span:d.value?24-R.value:24,class:"left"},{default:e(()=>[t(T,null,{default:e(()=>[t(p,{justify:"space-between",style:{width:"100%"}},{default:e(()=>[t(_,{level:3},{default:e(()=>[a(" 基础信息 ")]),_:1}),t(c,{type:"text",style:{color:"#0a90d5"},onClick:r[0]||(r[0]=u=>d.value=!d.value)},{default:e(()=>[a(n(o.$t("flowControlDomain.versionRecords"))+" ",1),d.value?(i(),m(w(Q),{key:1})):(i(),m(w(K),{key:0}))]),_:1})]),_:1}),t(g,{class:"_detail"},{default:e(()=>[t(z,{column:2,layout:"vertical",title:""},{default:e(()=>[t(f,{label:o.$t("flowControlDomain.ruleName"),labelStyle:{fontWeight:"bold"}},{default:e(()=>[C("p",{class:"description-item-content with-card",onClick:r[1]||(r[1]=u=>V(s.key))},[a(n(s.key)+" ",1),t(w(B))])]),_:1},8,["label"]),t(f,{label:o.$t("flowControlDomain.ruleGranularity"),labelStyle:{fontWeight:"bold"}},{default:e(()=>[t(y,null,{default:e(()=>[a(n(s.scope),1)]),_:1})]),_:1},8,["label"]),t(f,{label:o.$t("flowControlDomain.actionObject"),labelStyle:{fontWeight:"bold"}},{default:e(()=>[C("p",{class:"description-item-content with-card",onClick:r[2]||(r[2]=u=>V(I.value))},[a(n(I.value)+" ",1),t(w(B))])]),_:1},8,["label"]),t(f,{label:o.$t("flowControlDomain.faultTolerantProtection"),labelStyle:{fontWeight:"bold"}},{default:e(()=>[t(y,null,{default:e(()=>[a(n(s.force?o.$t("flowControlDomain.closed"):o.$t("flowControlDomain.opened")),1)]),_:1})]),_:1},8,["label"]),t(f,{label:o.$t("flowControlDomain.enabledState"),labelStyle:{fontWeight:"bold"}},{default:e(()=>[t(y,null,{default:e(()=>[a(n(s.enabled?o.$t("flowControlDomain.enabled"):o.$t("flowControlDomain.disabled")),1)]),_:1})]),_:1},8,["label"]),t(f,{label:o.$t("flowControlDomain.runTimeEffective"),labelStyle:{fontWeight:"bold"}},{default:e(()=>[t(y,null,{default:e(()=>[a(n(s.runtime?o.$t("flowControlDomain.opened"):o.$t("flowControlDomain.closed")),1)]),_:1})]),_:1},8,["label"])]),_:1})]),_:1})]),_:1}),t(g,{style:{"margin-top":"10px"},class:"_detail"},{default:e(()=>[t($,{align:"start",style:{width:"100%"}},{default:e(()=>[t(_,{level:5},{default:e(()=>[a(n(o.$t("flowControlDomain.requestParameterMatching"))+": ",1)]),_:1}),t($,{align:"center",direction:"horizontal",size:"middle"},{default:e(()=>[(i(!0),v(S,null,O(D.value,(u,k)=>(i(),m(N,{key:k,color:"#2db7f5"},{default:e(()=>[a(n(u),1)]),_:2},1024))),128))]),_:1})]),_:1}),t($,{align:"start",style:{width:"100%"}},{default:e(()=>[t(_,{level:5},{default:e(()=>[a(n(o.$t("flowControlDomain.addressSubsetMatching"))+": ",1)]),_:1}),(i(!0),v(S,null,O(W.value,(u,k)=>(i(),m(N,{key:k,color:"#87d068"},{default:e(()=>[a(n(u),1)]),_:2},1024))),128))]),_:1})]),_:1})]),_:1},8,["span"]),t(j,{span:d.value?R.value:0,class:"right"},{default:e(()=>[d.value?(i(),m(g,{key:0,class:"sliderBox"},{default:e(()=>[(i(),v(S,null,O(2,u=>t(g,{key:u},{default:e(()=>[le,ne,t(p,{justify:"flex-end"},{default:e(()=>[t(c,{type:"text",style:{color:"#0a90d5"}},{default:e(()=>[a("查看")]),_:1}),t(c,{type:"text",style:{color:"#0a90d5"}},{default:e(()=>[a("回滚")]),_:1})]),_:1})]),_:2},1024)),64))]),_:1})):X("",!0)]),_:1},8,["span"])]),_:1})])}}}),ce=te(se,[["__scopeId","data-v-b91b849f"]]);export{ce as default};