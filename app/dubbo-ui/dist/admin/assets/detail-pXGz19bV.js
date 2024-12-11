import{u as K}from"./index-XCC-U0tz.js";import{d as Q,u as U,b as X,r as L,g as Z,R as x,O as D,f,c as o,t,e as n,o as u,H as y,y as r,z as l,h as b,V as h,v as S,J as P,F as A,x as E,W as ee,_ as oe}from"./index-7CzEMIL1.js";import{g as te}from"./instance-D6AyE4yo.js";import{f as W}from"./DateUtil-AeksNPuC.js";import"./request-SYZV5A5R.js";const le={class:"__container_instance_detail"},ae={class:"white_space"},se={class:"white_space"},re={class:"white_space"},de=Q({__name:"detail",setup(pe){const k=U(),F=X(),v=L({}),{appContext:{config:{globalProperties:M}}}=Z();x("20");const e=L({});D(async()=>{let a={instanceName:k.params.pathId};v.detail=await te(a),Object.assign(e,v.detail.data),console.log("assign",e)});const j=()=>{F.push({path:"/resources/applications/detail",params:{pathId:k.params.pathId}})},z=K().toClipboard;function c(a){ee.success(M.$t("messageDomain.success.copy")),z(a)}const $=a=>a?"开启":"关闭";return(a,p)=>{const s=n("a-descriptions-item"),_=n("a-typography-paragraph"),C=n("a-descriptions"),m=n("a-card"),w=n("a-col"),H=n("a-row"),I=n("a-typography-link"),J=n("a-space"),Y=n("a-tag"),q=n("a-card-grid"),G=n("a-flex");return u(),f("div",le,[o(G,null,{default:t(()=>[o(q,null,{default:t(()=>[o(H,{gutter:10},{default:t(()=>[o(w,{span:12},{default:t(()=>[o(m,{class:"_detail"},{default:t(()=>[o(C,{class:"description-column",column:1},{default:t(()=>[o(s,{label:a.$t("instanceDomain.instanceName"),labelStyle:{fontWeight:"bold"}},{default:t(()=>{var d;return[y("p",{onClick:p[0]||(p[0]=i=>{var g;return c((g=b(k).params)==null?void 0:g.pathId)}),class:"description-item-content with-card"},[r(l((d=b(k).params)==null?void 0:d.pathId)+" ",1),o(b(h))])]}),_:1},8,["label"]),o(s,{label:a.$t("instanceDomain.creationTime_k8s"),labelStyle:{fontWeight:"bold"}},{default:t(()=>[o(_,null,{default:t(()=>[r(l(b(W)(e==null?void 0:e.createTime)),1)]),_:1})]),_:1},8,["label"]),o(s,{label:a.$t("instanceDomain.deployState"),labelStyle:{fontWeight:"bold"}},{default:t(()=>[(e==null?void 0:e.deployState)==="Running"?(u(),S(_,{key:0,type:"success"},{default:t(()=>[r(" Running ")]),_:1})):(u(),S(_,{key:1,type:"danger"},{default:t(()=>[r(" Stop")]),_:1}))]),_:1},8,["label"])]),_:1})]),_:1})]),_:1}),o(w,{span:12},{default:t(()=>[o(m,{class:"_detail"},{default:t(()=>[o(C,{class:"description-column",column:1},{default:t(()=>[o(s,{label:a.$t("instanceDomain.startTime_k8s"),labelStyle:{fontWeight:"bold"}},{default:t(()=>[o(_,null,{default:t(()=>[r(l(b(W)(e==null?void 0:e.readyTime)),1)]),_:1})]),_:1},8,["label"]),o(s,{label:a.$t("instanceDomain.registerState"),labelStyle:{fontWeight:"bold"}},{default:t(()=>[o(_,{type:(e==null?void 0:e.registerState)==="Registed"?"success":"danger"},{default:t(()=>[r(l(e==null?void 0:e.registerState),1)]),_:1},8,["type"])]),_:1},8,["label"]),o(s,{label:a.$t("instanceDomain.registerTime"),labelStyle:{fontWeight:"bold"}},{default:t(()=>[o(_,null,{default:t(()=>[r(l(b(W)(e==null?void 0:e.registerTime)),1)]),_:1})]),_:1},8,["label"])]),_:1})]),_:1})]),_:1})]),_:1}),o(m,{style:{"margin-top":"10px"},class:"_detail"},{default:t(()=>[o(C,{class:"description-column",column:1},{default:t(()=>[o(s,{label:a.$t("instanceDomain.instanceIP"),labelStyle:{fontWeight:"bold"}},{default:t(()=>[y("p",{onClick:p[1]||(p[1]=d=>c(e==null?void 0:e.ip)),class:"description-item-content with-card"},[r(l(e==null?void 0:e.ip)+" ",1),o(b(h))])]),_:1},8,["label"]),o(s,{label:a.$t("instanceDomain.deployCluster"),labelStyle:{fontWeight:"bold"}},{default:t(()=>[o(_,null,{default:t(()=>[r(l(e==null?void 0:e.deployCluster),1)]),_:1})]),_:1},8,["label"]),o(s,{label:a.$t("instanceDomain.dubboPort"),labelStyle:{fontWeight:"bold"}},{default:t(()=>[e!=null&&e.rpcPort?(u(),f("p",{key:0,onClick:p[2]||(p[2]=d=>c(e==null?void 0:e.rpcPort)),class:"description-item-content with-card"},[r(l(e==null?void 0:e.rpcPort)+" ",1),o(b(h))])):P("",!0)]),_:1},8,["label"]),o(s,{label:a.$t("instanceDomain.registerCluster"),labelStyle:{fontWeight:"bold"}},{default:t(()=>[o(J,null,{default:t(()=>[(u(!0),f(A,null,E(e==null?void 0:e.registerClusters,d=>(u(),S(I,null,{default:t(()=>[r(l(d),1)]),_:2},1024))),256))]),_:1})]),_:1},8,["label"]),o(s,{label:a.$t("instanceDomain.whichApplication"),labelStyle:{fontWeight:"bold"}},{default:t(()=>[o(I,{onClick:p[3]||(p[3]=d=>j())},{default:t(()=>[r(l(e==null?void 0:e.appName),1)]),_:1})]),_:1},8,["label"]),o(s,{label:a.$t("instanceDomain.node"),labelStyle:{fontWeight:"bold"}},{default:t(()=>[(e==null?void 0:e.node.length)>0?(u(),f("p",{key:0,onClick:p[4]||(p[4]=d=>c(e==null?void 0:e.node)),class:"description-item-content with-card"},[r(l(e==null?void 0:e.node)+" ",1),o(b(h))])):P("",!0)]),_:1},8,["label"]),o(s,{label:a.$t("instanceDomain.owningWorkload_k8s"),labelStyle:{fontWeight:"bold"}},{default:t(()=>[o(_,null,{default:t(()=>[r(l(e==null?void 0:e.workloadName),1)]),_:1})]),_:1},8,["label"]),o(s,{label:a.$t("instanceDomain.instanceImage_k8s"),labelStyle:{fontWeight:"bold"}},{default:t(()=>[o(m,{class:"description-item-card"},{default:t(()=>[(e==null?void 0:e.image.length)>0?(u(),f("p",{key:0,onClick:p[5]||(p[5]=d=>c(e==null?void 0:e.image)),class:"description-item-content with-card"},[r(l(e==null?void 0:e.image)+" ",1),o(b(h))])):P("",!0)]),_:1})]),_:1},8,["label"]),o(s,{label:a.$t("instanceDomain.instanceLabel"),labelStyle:{fontWeight:"bold"}},{default:t(()=>[o(m,{class:"description-item-card"},{default:t(()=>[(u(!0),f(A,null,E(e==null?void 0:e.labels,(d,i)=>(u(),S(Y,null,{default:t(()=>[r(l(i)+" : "+l(d),1)]),_:2},1024))),256))]),_:1})]),_:1},8,["label"]),o(s,{label:a.$t("instanceDomain.healthExamination_k8s"),labelStyle:{fontWeight:"bold"}},{default:t(()=>[o(m,{class:"description-item-card"},{default:t(()=>{var d,i,g,R,N,T,O,V,B;return[y("p",ae," 启动探针(StartupProbe):"+l($((d=e==null?void 0:e.probes)==null?void 0:d.startupProbe.open))+" 类型: "+l((i=e==null?void 0:e.probes)==null?void 0:i.startupProbe.type)+" 端口:"+l((g=e==null?void 0:e.probes)==null?void 0:g.startupProbe.port),1),y("p",se," 就绪探针(ReadinessProbe):"+l($((R=e==null?void 0:e.probes)==null?void 0:R.readinessProbe.open))+" 类型: "+l((N=e==null?void 0:e.probes)==null?void 0:N.readinessProbe.type)+" 端口:"+l((T=e==null?void 0:e.probes)==null?void 0:T.readinessProbe.port),1),y("p",re," 存活探针(LivenessProbe):"+l($((O=e==null?void 0:e.probes)==null?void 0:O.livenessProbe.open))+" 类型: "+l((V=e==null?void 0:e.probes)==null?void 0:V.livenessProbe.type)+" 端口:"+l((B=e==null?void 0:e.probes)==null?void 0:B.livenessProbe.port),1)]}),_:1})]),_:1},8,["label"])]),_:1})]),_:1})]),_:1})]),_:1})])}}}),fe=oe(de,[["__scopeId","data-v-747e722d"]]);export{fe as default};
