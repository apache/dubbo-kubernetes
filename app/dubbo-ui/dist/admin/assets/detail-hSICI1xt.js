import{d as V,u as W,r as N,g as B,R as T,O as A,f as s,c as a,t as e,e as c,o,F as u,x as m,U as P,v as b,y as S,z as h,h as y,V as F,W as L,X as I,_ as j}from"./index-0c4Jk4Xf.js";import{g as z}from"./app-fBErW7re.js";import{u as E}from"./index-Haky1Klh.js";import"./request-fr6f-AQz.js";const U={class:"__container_app_detail"},X={class:"description-item-content no-card"},Y=["onClick"],q=V({__name:"detail",setup(G){const R=W(),C=N({}),{appContext:{config:{globalProperties:$}}}=B();T("20");let r=N({left:{},right:{},bottom:{}});A(async()=>{var i;let l=(i=R.params)==null?void 0:i.pathId;C.detail=await z({appName:l}),console.log(C.detail);let{appName:x,rpcProtocols:p,dubboVersions:_,dubboPorts:d,serialProtocols:f,appTypes:w,images:k,workloads:D,deployClusters:t,registerClusters:n,registerModes:g}=C.detail.data;r.left={appName:x,appTypes:w,serialProtocols:f},r.right={rpcProtocols:p,dubboPorts:d,dubboVersions:_},r.bottom={images:k,workloads:D,deployClusters:t,registerClusters:n,registerModes:g},console.log(x)});const M=E().toClipboard;function O(l){L.success($.$t("messageDomain.success.copy")),M(l)}return(l,x)=>{const p=c("a-descriptions-item"),_=c("a-descriptions"),d=c("a-card"),f=c("a-col"),w=c("a-row"),k=c("a-card-grid"),D=c("a-flex");return o(),s("div",U,[a(D,null,{default:e(()=>[a(k,null,{default:e(()=>[a(w,{gutter:10},{default:e(()=>[a(f,{span:12},{default:e(()=>[a(d,{class:"_detail"},{default:e(()=>[a(_,{class:"description-column",column:1},{default:e(()=>[(o(!0),s(u,null,m(y(r).left,(t,n,g)=>P((o(),b(p,{labelStyle:{fontWeight:"bold",width:"100px"},label:l.$t("applicationDomain."+n)},{default:e(()=>[S(h(typeof t=="object"?t[0]:t),1)]),_:2},1032,["label"])),[[I,!!t]])),256))]),_:1})]),_:1})]),_:1}),a(f,{span:12},{default:e(()=>[a(d,{class:"_detail"},{default:e(()=>[a(_,{class:"description-column",column:1},{default:e(()=>[(o(!0),s(u,null,m(y(r).right,(t,n,g)=>P((o(),b(p,{labelStyle:{fontWeight:"bold",width:"100px"},label:l.$t("applicationDomain."+n)},{default:e(()=>[S(h(t[0]),1)]),_:2},1032,["label"])),[[I,!!t]])),256))]),_:1})]),_:1})]),_:1})]),_:1}),a(d,{style:{"margin-top":"10px"},class:"_detail"},{default:e(()=>[a(_,{class:"description-column",column:1},{default:e(()=>[(o(!0),s(u,null,m(y(r).bottom,(t,n,g)=>P((o(),b(p,{labelStyle:{fontWeight:"bold"},label:l.$t("applicationDomain."+n)},{default:e(()=>[(t==null?void 0:t.length)<3?(o(!0),s(u,{key:0},m(t,i=>(o(),s("p",X,h(i),1))),256)):(o(),b(d,{key:1,class:"description-item-card"},{default:e(()=>[(o(!0),s(u,null,m(t,i=>(o(),s("p",{onClick:H=>O(i.toString()),class:"description-item-content with-card"},[S(h(i)+" ",1),a(y(F))],8,Y))),256))]),_:2},1024))]),_:2},1032,["label"])),[[I,!!t]])),256))]),_:1})]),_:1})]),_:1})]),_:1})])}}}),v=j(q,[["__scopeId","data-v-2ded24e2"]]);export{v as default};