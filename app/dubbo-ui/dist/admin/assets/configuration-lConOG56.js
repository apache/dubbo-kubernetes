import{C as f}from"./ConfigPage-lUfTaFc5.js";import{d as m,r as d,O as _,f as g,c as a,t as l,h as p,e as c,o as u,_ as b}from"./index-7gOCAPeR.js";const h={class:"__container_app_config"},w=m({__name:"configuration",setup(k){let r=d({list:[{title:"instanceDomain.operatorLog",key:"log",form:{logFlag:!1},submit:e=>new Promise(t=>{setTimeout(()=>{t(1)},1e3)}),reset(e){e.logFlag=!1}},{title:"instanceDomain.flowDisabled",form:{flowDisabledFlag:!1},key:"flowDisabled",submit:e=>new Promise(t=>{setTimeout(()=>{t(1)},1e3)}),reset(e){e.logFlag=!1}}],current:[0]});return _(()=>{console.log(333)}),(e,t)=>{const s=c("a-switch"),i=c("a-form-item");return u(),g("div",h,[a(f,{options:p(r)},{form_log:l(({current:o})=>[a(i,{label:e.$t("instanceDomain.operatorLog"),name:"logFlag"},{default:l(()=>[a(s,{checked:o.form.logFlag,"onUpdate:checked":n=>o.form.logFlag=n},null,8,["checked","onUpdate:checked"])]),_:2},1032,["label"])]),form_flowDisabled:l(({current:o})=>[a(i,{label:e.$t("instanceDomain.flowDisabled"),name:"flowDisabledFlag"},{default:l(()=>[a(s,{checked:o.form.flowDisabledFlag,"onUpdate:checked":n=>o.form.flowDisabledFlag=n},null,8,["checked","onUpdate:checked"])]),_:2},1032,["label"])]),_:1},8,["options"])])}}}),C=b(w,[["__scopeId","data-v-a667a5b6"]]);export{C as default};