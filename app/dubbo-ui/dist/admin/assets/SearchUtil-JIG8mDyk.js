var K=Object.defineProperty;var W=(d,o,n)=>o in d?K(d,o,{enumerable:!0,configurable:!0,writable:!0,value:n}):d[o]=n;var _=(d,o,n)=>(W(d,typeof o!="symbol"?o+"":o,n),n);import{d as G,k as Q,r as A,g as Z,l as q,m as ee,a as D,f as m,H as h,c,t as r,y as S,z as g,h as l,e as u,P as te,o as i,F as y,x as w,v as f,ac as E,I as $,a3 as V,U as ae,X as oe,ad as le,W as se,_ as ne}from"./index-7gOCAPeR.js";const re={class:"__container_search_table"},ie={class:"search-query-container"},ce={class:"custom-column button"},ue={class:"dropdown"},de={class:"body"},_e=["onClick"],pe={class:"search-table-container"},me={key:0},he=G({__name:"SearchTable",setup(d){Q(t=>({"19905e97":l(te)}));const o=A({customColumns:!1}),{appContext:{config:{globalProperties:n}}}=Z(),e=q(ee.SEARCH_DOMAIN);e.table.columns.forEach(t=>{if(t.title){const p=t.title;t.title=D(()=>n.$t(p))}});const v=D(()=>e.noPaged?!1:{pageSize:e.paged.pageSize,current:e.paged.curPage,showTotal:t=>n.$t("searchDomain.total")+": "+t+" "+n.$t("searchDomain.unit")}),k=(t,p,C)=>{e.paged.pageSize=t.pageSize,e.paged.curPage=t.current,e.onSearch()};function b(t){let p=e==null?void 0:e.table.columns.filter(C=>!C.__hide);if(!t.__hide&&p.length<=1){se.warn("must show at least one column");return}t.__hide=!t.__hide}return(t,p)=>{var N,I,O,U;const C=u("a-radio-button"),B=u("a-radio-group"),R=u("a-select-option"),M=u("a-select"),Y=u("a-input"),x=u("a-form-item"),j=u("a-button"),z=u("a-flex"),H=u("a-form"),P=u("a-col"),J=u("a-card"),L=u("a-row"),X=u("a-table");return i(),m("div",re,[h("div",ie,[c(L,null,{default:r(()=>[c(P,{span:18},{default:r(()=>[c(H,null,{default:r(()=>[c(z,{wrap:"wrap",gap:"large"},{default:r(()=>[(i(!0),m(y,null,w(l(e).params,a=>(i(),f(x,{label:t.$t(a.label)},{default:r(()=>[a.dict&&a.dict.length>0?(i(),m(y,{key:0},[a.dictType==="BUTTON"?(i(),f(B,{key:0,"button-style":"solid",value:l(e).queryForm[a.param],"onUpdate:value":s=>l(e).queryForm[a.param]=s},{default:r(()=>[(i(!0),m(y,null,w(a.dict,s=>(i(),f(C,{value:s.value},{default:r(()=>[S(g(t.$t(s.label)),1)]),_:2},1032,["value"]))),256))]),_:2},1032,["value","onUpdate:value"])):(i(),f(M,{key:1,class:"select-type",style:E(a.style),value:l(e).queryForm[a.param],"onUpdate:value":s=>l(e).queryForm[a.param]=s},{default:r(()=>[(i(!0),m(y,null,w([...a.dict,{label:"none",value:""}],s=>(i(),f(R,{value:s.value},{default:r(()=>[S(g(t.$t(s.label)),1)]),_:2},1032,["value"]))),256))]),_:2},1032,["style","value","onUpdate:value"]))],64)):(i(),f(Y,{key:1,style:E(a.style),placeholder:t.$t("placeholder."+(a.placeholder||"typeDefault")),value:l(e).queryForm[a.param],"onUpdate:value":s=>l(e).queryForm[a.param]=s},null,8,["style","placeholder","value","onUpdate:value"]))]),_:2},1032,["label"]))),256)),c(x,{label:""},{default:r(()=>[c(j,{type:"primary",onClick:p[0]||(p[0]=a=>l(e).onSearch())},{default:r(()=>[c(l($),{style:{"margin-bottom":"-2px","font-size":"1.3rem"},icon:"ic:outline-manage-search"})]),_:1})]),_:1})]),_:1})]),_:1})]),_:1}),c(P,{span:6},{default:r(()=>[c(z,{style:{"justify-content":"flex-end"}},{default:r(()=>[V(t.$slots,"customOperation",{},void 0,!0),h("div",{class:"common-tool",onClick:p[1]||(p[1]=a=>o.customColumns=!o.customColumns)},[h("div",ce,[c(l($),{icon:"material-symbols-light:format-list-bulleted-rounded"})]),ae(h("div",ue,[c(J,{style:{"max-width":"300px"},title:"Custom Column"},{default:r(()=>{var a;return[h("div",de,[(i(!0),m(y,null,w((a=l(e))==null?void 0:a.table.columns,(s,F)=>(i(),m("div",{class:"item",onClick:le(T=>b(s),["stop"])},[c(l($),{style:{"margin-bottom":"-4px","font-size":"1rem","margin-right":"2px"},icon:s.__hide?"zondicons:view-hide":"zondicons:view-show"},null,8,["icon"]),S(" "+g(s.title),1)],8,_e))),256))])]}),_:1})],512),[[oe,o.customColumns]])])]),_:3})]),_:3})]),_:3})]),h("div",pe,[S(g(JSON.stringify(l(e)))+" ",1),c(X,{loading:l(e).table.loading,pagination:v.value,scroll:{scrollToFirstRowOnChange:!0,y:((N=l(e).tableStyle)==null?void 0:N.scrollY)||"",x:((I=l(e).tableStyle)==null?void 0:I.scrollX)||""},columns:(O=l(e))==null?void 0:O.table.columns.filter(a=>!a.__hide),"data-source":(U=l(e))==null?void 0:U.result,onChange:k},{bodyCell:r(({text:a,record:s,index:F,column:T})=>[T.key==="idx"?(i(),m("span",me,g(F+1),1)):V(t.$slots,"bodyCell",{key:1,text:a,record:s,index:F,column:T},void 0,!0)]),_:3},8,["loading","pagination","scroll","columns","data-source"])])])}}}),ye=ne(he,[["__scopeId","data-v-a33c25a5"]]);class ve{constructor(o,n,e,v,k,b){_(this,"noPaged");_(this,"queryForm");_(this,"params");_(this,"searchApi");_(this,"result");_(this,"handleResult");_(this,"tableStyle");_(this,"table",{columns:[]});_(this,"paged",{curPage:1,total:0,pageSize:10});this.params=o,this.noPaged=k,this.queryForm=A({}),this.table.columns=e,o.forEach(t=>{t.defaultValue&&(this.queryForm[t.param]=t.defaultValue)}),v&&(this.paged={...this.paged,...v}),this.searchApi=n,b&&this.onSearch(b)}async onSearch(o){this.table.loading=!0,setTimeout(()=>{this.table.loading=!1},5e3);const n=(await this.searchApi(this.queryForm||{})).data;this.result=o?o(n.data):n,console.log(this.result),this.paged.total=n.total,this.table.loading=!1}}function be(d,o){return isNaN(d-o)?d.localeCompare(o):d-o}export{ve as S,ye as a,be as s};