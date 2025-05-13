import{y as b,_ as A}from"./js-yaml-51IX0JDU.js";import{d as D,l as S,m as T,b as O,n as f,Q as Y,s as $,v as p,t as e,e as s,o as g,c as t,z as a,h as w,a6 as L,a7 as M,J as o,K as P,G as J,H as K,_ as z}from"./index-2tg1dFrv.js";import{a as j}from"./traffic-uK9CJKqJ.js";import"./request-2UPhfKnX.js";const d=_=>(J("data-v-f528faad"),_=_(),K(),_),G={class:"editorBox"},H={class:"bottom-action-footer"},Q=d(()=>o("br",null,null,-1)),U=d(()=>o("br",null,null,-1)),q=d(()=>o("br",null,null,-1)),F=d(()=>o("br",null,null,-1)),W=d(()=>o("br",null,null,-1)),X=d(()=>o("br",null,null,-1)),Z=D({__name:"addByYAMLView",setup(_){const h=S(T.PROVIDE_INJECT_KEY),E=O(),I=f(!1),l=f(!1),y=f(8),r=f(`conditions:
  - from:
      match: >-
        method=string & arguments[method]=string &
        arguments[arguments[method]]=string &
        arguments[arguments[arguments[method]]]=string &
        arguments[arguments[arguments[arguments[string]]]]!=string
    to:
      - match: string!=string
        weight: 0
  - from:
      match: >-
        method=string & arguments[method]=string &
        arguments[arguments[method]]=string &
        arguments[arguments[arguments[string]]]!=string
    to:
      - match: string!=lggbond
        weight: 0
      - match: ss!=ss
        weight: 0
configVersion: v3.1
enabled: true
force: false
key: org.apache.dubbo.samples.CommentService
runtime: true
scope: service`);Y(()=>{if($.isNil(h.conditionRule))r.value="";else{const i=h.conditionRule;r.value=b.dump(i)}});const B=i=>{h.conditionRule=b.load(r.value)},N=async()=>{const i=b.load(r.value),{configVersion:u,scope:m,key:c,runtime:V,force:x,conditions:v}=i;let n="";c=="application"?n=`${c}.condition-router`:n=`${c}:${u}.condition-router`,(await j(n,i)).code===200&&E.push("/traffic/routingRule")};return(i,u)=>{const m=s("a-button"),c=s("a-flex"),V=s("a-space"),x=s("a-affix"),v=s("a-col"),n=s("a-descriptions-item"),R=s("a-descriptions"),C=s("a-card");return g(),p(C,null,{default:e(()=>[t(c,{style:{width:"100%"}},{default:e(()=>[t(v,{span:l.value?24-y.value:24,class:"left"},{default:e(()=>[t(c,{vertical:"",align:"end"},{default:e(()=>[t(m,{type:"text",style:{color:"#0a90d5"},onClick:u[0]||(u[0]=k=>l.value=!l.value)},{default:e(()=>[a(" 字段说明 "),l.value?(g(),p(w(M),{key:1})):(g(),p(w(L),{key:0}))]),_:1}),o("div",G,[t(A,{onChange:B,modelValue:r.value,"onUpdate:modelValue":u[1]||(u[1]=k=>r.value=k),theme:"vs-dark",height:500,language:"yaml",readonly:I.value},null,8,["modelValue","readonly"])])]),_:1}),t(x,{"offset-bottom":10},{default:e(()=>[o("div",H,[t(V,{align:"center",size:"large"},{default:e(()=>[t(m,{type:"primary",onClick:N},{default:e(()=>[a(" 确认 ")]),_:1}),t(m,null,{default:e(()=>[a(" 取消 ")]),_:1})]),_:1})])]),_:1})]),_:1},8,["span"]),t(v,{span:l.value?y.value:0,class:"right"},{default:e(()=>[l.value?(g(),p(C,{key:0,class:"sliderBox"},{default:e(()=>[o("div",null,[t(R,{title:"字段说明",column:1},{default:e(()=>[t(n,{label:"key"},{default:e(()=>[a(" 作用对象"),Q,a(" 可能的值：Dubbo应用名或者服务名 ")]),_:1}),t(n,{label:"scope"},{default:e(()=>[a(" 规则粒度"),U,a(" 可能的值：application, service ")]),_:1}),t(n,{label:"force"},{default:e(()=>[a(" 容错保护"),q,a(" 可能的值：true, false"),F,a(" 描述：如果为true，则路由筛选后若没有可用的地址则会直接报异常；如果为false，则会从可用地址中选择完成RPC调用 ")]),_:1}),t(n,{label:"runtime"},{default:e(()=>[a(" 运行时生效"),W,a(" 可能的值：true, false"),X,a(" 描述：如果为true，则该rule下的所有路由将会实时生效；若为false，则只有在启动时才会生效 ")]),_:1})]),_:1})])]),_:1})):P("",!0)]),_:1},8,["span"])]),_:1})]),_:1})}}}),ne=z(Z,[["__scopeId","data-v-f528faad"]]);export{ne as default};
