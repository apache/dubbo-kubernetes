import{C as H}from"./ConfigPage-Ie-50VFr.js";import{d as X,u as Y,r as Z,a0 as ee,Q as ae,f as I,c as l,t as o,h as W,e as _,o as p,F as $,y as G,v as g,z as C,B as U,J as L,I as z,K as w,_ as te}from"./index-WfxyH1kR.js";import{u as oe,e as le,f as ne,h as se,i as ie,j as ue}from"./app-6wT5F1TY.js";import"./request-2cLtQ4vm.js";const ce=D=>{var m;(m=document.getElementById(D))==null||m.scrollIntoView({behavior:"smooth"})},re={class:"__container_app_config"},de={style:{float:"right"}},fe={style:{float:"right"}},pe=X({__name:"config",setup(D){const m=Y(),A=[{key:"key",title:"label"},{key:"condition",title:"condition"},{key:"value",title:"value"},{key:"operation",title:"操作"}];let r=Z({list:[{title:"applicationDomain.operatorLog",key:"log",form:{logFlag:!1},submit:e=>new Promise(a=>{a(B(e==null?void 0:e.logFlag))}),reset(e){e.logFlag=!1}},{title:"applicationDomain.flowWeight",key:"flow",ext:{title:"添加权重配置",fun(e){O(),ee(()=>{var t;const a=((t=r.list.find(n=>n.key==="flow"))==null?void 0:t.form.rules.length)-1;a>=0&&ce("flowWeight"+a)})}},form:{rules:[{weight:10,scope:[]}]},submit(e){return new Promise(a=>{a(N())})},reset(){x()}},{title:"applicationDomain.gray",key:"gray",ext:{title:"添加灰度环境",fun(){M()}},form:{rules:[{name:"env-nam",scope:{key:"env",value:{exact:"gray"}}}]},submit(e){return new Promise(a=>{a(K())})},reset(){F()}}],current:[0]});const T=async()=>{var a;const e=await ue((a=m.params)==null?void 0:a.pathId);console.log(e),(e==null?void 0:e.code)==200&&r.list.forEach(t=>{if(t.key==="log"){t.form.logFlag=e.data.operatorLog;return}})},B=async e=>{var t;const a=await oe((t=m.params)==null?void 0:t.pathId,e);console.log(a),(a==null?void 0:a.code)==200&&await T()},x=async()=>{var a;const e=await le((a=m.params)==null?void 0:a.pathId);(e==null?void 0:e.code)==200&&r.list.forEach(t=>{t.key==="flow"&&(t.form.rules=JSON.parse(JSON.stringify(e.data.flowWeightSets)),t.form.rules.forEach(n=>{n.scope.forEach(s=>{s.label=s.key,s.condition=s.value?Object.keys(s.value)[0]:"",s.value=s.value?Object.values(s.value)[0]:""})}))})},N=async()=>{var t;let e=[];r.list.forEach(n=>{n.key==="flow"&&n.form.rules.forEach(s=>{let i={weight:s.weight,scope:[]};s.scope.forEach(b=>{const{key:v,value:E,condition:h}=b;let y={key:b.label||v,value:{}};h&&(y.value[h]=E),i.scope.push(y)}),e.push(i)})}),(await ne((t=m.params)==null?void 0:t.pathId,e)).code===200&&await x()},O=()=>{r.list.forEach(e=>{e.key==="flow"&&e.form.rules.push({weight:10,scope:[{key:"",condition:"",value:""}]})})},V=e=>{r.list.forEach(a=>{a.key==="flow"&&a.form.rules.splice(e,1)})},j=e=>{r.list.forEach(a=>{if(a.key==="flow"){let t={key:"",condition:"",value:""};a.form.rules[e].scope.push(t);return}})},P=(e,a)=>{r.list.forEach(t=>{t.key==="flow"&&t.form.rules[e].scope.splice(a,1)})},J=[{key:"label",title:"label",dataIndex:"label"},{key:"condition",title:"condition",dataIndex:"condition"},{key:"value",title:"value",dataIndex:"value"},{key:"operation",title:"operation",dataIndex:"operation"}],F=async()=>{var a;const e=await se((a=m.params)==null?void 0:a.pathId);(e==null?void 0:e.code)==200&&r.list.forEach(t=>{if(t.key==="gray"){const n=e.data.graySets;n.length>0&&n.forEach(s=>{s.scope.forEach(i=>{i.label=i.key,i.condition=i.value?Object.keys(i.value)[0]:"",i.value=i.value?Object.values(i.value)[0]:""})}),t.form.rules=n}})},K=async()=>{var t;let e=[];r.list.forEach(n=>{n.key==="gray"&&n.form.rules.forEach(s=>{let i={name:s.name,scope:[]};s.scope.forEach(b=>{const{key:v,value:E,condition:h}=b;let y={key:b.label,value:{}};h&&(y.value[h]=E),i.scope.push(y)}),e.push(i)})}),(await ie((t=m.params)==null?void 0:t.pathId,e)).code===200&&await F()},M=()=>{r.list.forEach(e=>{e.key==="gray"&&e.form.rules.push({name:"",scope:[{key:"",condition:"",value:""}]})})},Q=e=>{r.list.forEach(a=>{if(a.key==="gray"){let t={key:"",condition:"",value:""};a.form.rules[e].scope.push(t);return}})},R=(e,a)=>{r.list.forEach(t=>{t.key==="gray"&&t.form.rules[e].scope.splice(a,1)})},q=e=>{r.list.forEach(a=>{a.key==="gray"&&a.form.rules.splice(e,1)})};return ae(()=>{T(),x(),F()}),(e,a)=>{const t=_("a-switch"),n=_("a-form-item"),s=_("a-button"),i=_("a-space"),b=_("a-input-number"),v=_("a-input"),E=_("a-table"),h=_("a-card");return p(),I("div",re,[l(H,{options:W(r)},{form_log:o(({current:y})=>[l(n,{label:e.$t("applicationDomain.operatorLog"),name:"logFlag"},{default:o(()=>[l(t,{checked:y.form.logFlag,"onUpdate:checked":k=>y.form.logFlag=k},null,8,["checked","onUpdate:checked"])]),_:2},1032,["label"])]),form_flow:o(({current:y})=>[l(i,{direction:"vertical",size:"middle",class:"flowWeight-box"},{default:o(()=>[(p(!0),I($,null,G(y.form.rules,(k,u)=>(p(),g(h,{id:"flowWeight"+u},{title:o(()=>[C(U(e.$t("applicationDomain.flowWeight"))+" "+U(u+1)+" ",1),L("div",de,[l(i,null,{default:o(()=>[l(s,{danger:"",type:"dashed",onClick:c=>V(u)},{default:o(()=>[l(W(z),{style:{"font-size":"20px"},icon:"fluent:delete-12-filled"})]),_:2},1032,["onClick"])]),_:2},1024)])]),default:o(()=>[l(n,{name:"rules["+u+"].weight",label:"权重"},{default:o(()=>[l(b,{min:"1",value:k.weight,"onUpdate:value":c=>k.weight=c},null,8,["value","onUpdate:value"])]),_:2},1032,["name"]),l(n,{label:"作用范围"},{default:o(()=>[l(s,{type:"primary",onClick:c=>j(u)},{default:o(()=>[C(" 添加")]),_:2},1032,["onClick"]),l(E,{style:{width:"40vw"},pagination:!1,columns:A,"data-source":k.scope},{bodyCell:o(({column:c,record:d,index:S})=>[c.key==="key"?(p(),g(n,{key:0,name:"rules["+u+"].scope.key"},{default:o(()=>[l(v,{value:d.key,"onUpdate:value":f=>d.key=f},null,8,["value","onUpdate:value"])]),_:2},1032,["name"])):w("",!0),c.key==="condition"?(p(),g(n,{key:1,name:"rules["+u+"].scope.condition"},{default:o(()=>[l(v,{value:d.condition,"onUpdate:value":f=>d.condition=f},null,8,["value","onUpdate:value"])]),_:2},1032,["name"])):w("",!0),c.key==="value"?(p(),g(n,{key:2,name:"rules["+u+"].scope.value"},{default:o(()=>[l(v,{value:d.value,"onUpdate:value":f=>d.value=f},null,8,["value","onUpdate:value"])]),_:2},1032,["name"])):w("",!0),c.key==="operation"?(p(),g(n,{key:3,name:"rules["+u+"].scope.operation"},{default:o(()=>[l(s,{type:"link",onClick:f=>P(u,S)},{default:o(()=>[C(" 删除")]),_:2},1032,["onClick"])]),_:2},1032,["name"])):w("",!0)]),_:2},1032,["data-source"])]),_:2},1024)]),_:2},1032,["id"]))),256))]),_:2},1024)]),form_gray:o(({current:y})=>[l(i,{direction:"vertical",size:"middle"},{default:o(()=>[(p(!0),I($,null,G(y.form.rules,(k,u)=>(p(),g(h,null,{title:o(()=>[C(U(e.$t("applicationDomain.gray"))+" "+U(u+1)+" ",1),L("div",fe,[l(i,null,{default:o(()=>[l(s,{danger:"",type:"dashed",onClick:c=>q(u)},{default:o(()=>[l(W(z),{style:{"font-size":"20px"},icon:"fluent:delete-12-filled"})]),_:2},1032,["onClick"])]),_:2},1024)])]),default:o(()=>[l(n,{name:"rules["+u+"].name",label:"环境名称"},{default:o(()=>[l(v,{value:k.name,"onUpdate:value":c=>k.name=c},null,8,["value","onUpdate:value"])]),_:2},1032,["name"]),l(n,{label:"作用范围"},{default:o(()=>[l(i,{direction:"vertical",size:"middle"},{default:o(()=>[l(s,{type:"primary",onClick:c=>Q(u)},{default:o(()=>[C(" 添加")]),_:2},1032,["onClick"]),l(E,{style:{width:"40vw"},pagination:!1,columns:J,"data-source":k.scope},{bodyCell:o(({column:c,record:d,index:S})=>[c.key==="label"?(p(),g(n,{key:0,name:"rules["+u+"].scope.key"},{default:o(()=>[l(v,{value:d.label,"onUpdate:value":f=>d.label=f},null,8,["value","onUpdate:value"])]),_:2},1032,["name"])):w("",!0),c.key==="condition"?(p(),g(n,{key:1,name:"rules["+u+"].scope.condition"},{default:o(()=>[l(v,{value:d.condition,"onUpdate:value":f=>d.condition=f},null,8,["value","onUpdate:value"])]),_:2},1032,["name"])):w("",!0),c.key==="value"?(p(),g(n,{key:2,name:"rules["+u+"].scope.value"},{default:o(()=>[l(v,{value:d.value,"onUpdate:value":f=>d.value=f},null,8,["value","onUpdate:value"])]),_:2},1032,["name"])):w("",!0),c.key==="operation"?(p(),g(n,{key:3,name:"rules["+u+"].scope.operation"},{default:o(()=>[l(s,{type:"link",onClick:f=>R(u,S)},{default:o(()=>[C(" 删除")]),_:2},1032,["onClick"])]),_:2},1032,["name"])):w("",!0)]),_:2},1032,["data-source"])]),_:2},1024)]),_:2},1024)]),_:2},1024))),256))]),_:2},1024)]),_:1},8,["options"])])}}}),ke=te(pe,[["__scopeId","data-v-2bf9591a"]]);export{ke as default};
