import{d as f,k as x,b as g,r as b,f as i,c as p,t as c,h as d,P as C,K as N,e as v,o as s,H as w,I as R,y as _,z as m,v as h,F as T,x as V,J as A,m as D,_ as P}from"./index-7gOCAPeR.js";import{s as E}from"./service-Pa7FOyqK.js";import{S as B,a as O}from"./SearchUtil-JIG8mDyk.js";/* empty css                                                               */import"./request--Xss-qAk.js";const q={class:"__container_services_index"},Q=["onClick"],F=f({__name:"search",setup(H){x(r=>({"4600fa7a":d(C)}));const y=g(),G=[{title:"service",key:"service",dataIndex:"serviceName",sorter:!0,width:"30%"},{title:"versionGroup",key:"versionGroup",dataIndex:"versionGroupSelect",width:"25%"},{title:"avgQPS",key:"avgQPS",dataIndex:"avgQPS",sorter:!0,width:"15%"},{title:"avgRT",key:"avgRT",dataIndex:"avgRT",sorter:!0,width:"15%"},{title:"requestTotal",key:"requestTotal",dataIndex:"requestTotal",sorter:!0,width:"15%"}],l=r=>r.map(e=>(e.versionGroupSelect={},e.versionGroupSelect.versionGroupArr=e.versionGroup.map(a=>a.versionGroup=(a.version?"version: "+a.version+", ":"")+(a.group?"group: "+a.group:"")||"无"),e.versionGroupSelect.versionGroupValue=e.versionGroupSelect.versionGroupArr[0],e)),n=b(new B([{label:"serviceName",param:"serviceName",placeholder:"typeAppName",style:{width:"200px"}}],E,G,void 0,void 0,l));n.onSearch(l);const S=r=>{y.push({name:"distribution",params:{pathId:r}})};return N(D.SEARCH_DOMAIN,n),(r,e)=>{const a=v("a-select-option"),k=v("a-select");return s(),i("div",q,[p(O,{"search-domain":n},{bodyCell:c(({column:u,text:o})=>[u.dataIndex==="serviceName"?(s(),i("span",{key:0,class:"service-link",onClick:t=>S(o)},[w("b",null,[p(d(R),{style:{"margin-bottom":"-2px"},icon:"material-symbols:attach-file-rounded"}),_(" "+m(o),1)])],8,Q)):u.dataIndex==="versionGroupSelect"?(s(),h(k,{key:1,value:o.versionGroupValue,"onUpdate:value":t=>o.versionGroupValue=t,bordered:!1,style:{width:"80%"}},{default:c(()=>[(s(!0),i(T,null,V(o.versionGroupArr,(t,I)=>(s(),h(a,{value:t,key:I},{default:c(()=>[_(m(t),1)]),_:2},1032,["value"]))),128))]),_:2},1032,["value","onUpdate:value"])):A("",!0)]),_:1},8,["search-domain"])])}}}),Y=P(F,[["__scopeId","data-v-c34ab506"]]);export{Y as default};