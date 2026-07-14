(function(){const r=document.createElement("link").relList;if(r&&r.supports&&r.supports("modulepreload"))return;for(const s of document.querySelectorAll('link[rel="modulepreload"]'))o(s);new MutationObserver(s=>{for(const a of s)if(a.type==="childList")for(const l of a.addedNodes)l.tagName==="LINK"&&l.rel==="modulepreload"&&o(l)}).observe(document,{childList:!0,subtree:!0});function t(s){const a={};return s.integrity&&(a.integrity=s.integrity),s.referrerPolicy&&(a.referrerPolicy=s.referrerPolicy),s.crossOrigin==="use-credentials"?a.credentials="include":s.crossOrigin==="anonymous"?a.credentials="omit":a.credentials="same-origin",a}function o(s){if(s.ep)return;s.ep=!0;const a=t(s);fetch(s.href,a)}})();/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const W=(e,r,t=[])=>{const o=document.createElementNS("http://www.w3.org/2000/svg",e);return Object.keys(r).forEach(s=>{o.setAttribute(s,String(r[s]))}),t.length&&t.forEach(s=>{const a=W(...s);o.appendChild(a)}),o};var K=([e,r,t])=>W(e,r,t);/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const X=e=>Array.from(e.attributes).reduce((r,t)=>(r[t.name]=t.value,r),{}),Z=e=>typeof e=="string"?e:!e||!e.class?"":e.class&&typeof e.class=="string"?e.class.split(" "):e.class&&Array.isArray(e.class)?e.class:"",J=e=>e.flatMap(Z).map(t=>t.trim()).filter(Boolean).filter((t,o,s)=>s.indexOf(t)===o).join(" "),Q=e=>e.replace(/(\w)(\w*)(_|-|\s*)/g,(r,t,o)=>t.toUpperCase()+o.toLowerCase()),O=(e,{nameAttr:r,icons:t,attrs:o})=>{var V;const s=e.getAttribute(r);if(s==null)return;const a=Q(s),l=t[a];if(!l)return console.warn(`${e.outerHTML} icon name was not found in the provided icons object.`);const h=X(e),[f,L,b]=l,x={...L,"data-lucide":s,...o,...h},w=J(["lucide",`lucide-${s}`,h,o]);w&&Object.assign(x,{class:w});const C=K([f,x,b]);return(V=e.parentNode)==null?void 0:V.replaceChild(C,e)};/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const u={xmlns:"http://www.w3.org/2000/svg",width:24,height:24,viewBox:"0 0 24 24",fill:"none",stroke:"currentColor","stroke-width":2,"stroke-linecap":"round","stroke-linejoin":"round"};/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Y=["svg",u,[["path",{d:"M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ee=["svg",u,[["path",{d:"M17 7 7 17"}],["path",{d:"M17 17H7V7"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const te=["svg",u,[["path",{d:"M7 7h10v10"}],["path",{d:"M7 17 17 7"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const re=["svg",u,[["circle",{cx:"12",cy:"12",r:"10"}],["line",{x1:"12",x2:"12",y1:"8",y2:"12"}],["line",{x1:"12",x2:"12.01",y1:"16",y2:"16"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ne=["svg",u,[["circle",{cx:"12",cy:"12",r:"10"}],["path",{d:"m9 12 2 2 4-4"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const oe=["svg",u,[["circle",{cx:"12",cy:"12",r:"10"}],["polyline",{points:"12 6 12 12 16.5 12"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const se=["svg",u,[["rect",{width:"14",height:"14",x:"8",y:"8",rx:"2",ry:"2"}],["path",{d:"M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ae=["svg",u,[["rect",{width:"16",height:"16",x:"4",y:"4",rx:"2"}],["rect",{width:"6",height:"6",x:"9",y:"9",rx:"1"}],["path",{d:"M15 2v2"}],["path",{d:"M15 20v2"}],["path",{d:"M2 15h2"}],["path",{d:"M2 9h2"}],["path",{d:"M20 15h2"}],["path",{d:"M20 9h2"}],["path",{d:"M9 2v2"}],["path",{d:"M9 20v2"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ie=["svg",u,[["ellipse",{cx:"12",cy:"5",rx:"9",ry:"3"}],["path",{d:"M3 5V19A9 3 0 0 0 21 19V5"}],["path",{d:"M3 12A9 3 0 0 0 21 12"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ce=["svg",u,[["path",{d:"M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"}],["polyline",{points:"7 10 12 15 17 10"}],["line",{x1:"12",x2:"12",y1:"15",y2:"3"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const le=["svg",u,[["path",{d:"M10.733 5.076a10.744 10.744 0 0 1 11.205 6.575 1 1 0 0 1 0 .696 10.747 10.747 0 0 1-1.444 2.49"}],["path",{d:"M14.084 14.158a3 3 0 0 1-4.242-4.242"}],["path",{d:"M17.479 17.499a10.75 10.75 0 0 1-15.417-5.151 1 1 0 0 1 0-.696 10.75 10.75 0 0 1 4.446-5.143"}],["path",{d:"m2 2 20 20"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const de=["svg",u,[["path",{d:"M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0"}],["circle",{cx:"12",cy:"12",r:"3"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ue=["svg",u,[["path",{d:"M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"}],["circle",{cx:"10",cy:"16",r:"2"}],["path",{d:"m16 10-4.5 4.5"}],["path",{d:"m15 11 1 1"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const pe=["svg",u,[["path",{d:"M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"}],["path",{d:"M3 3v5h5"}],["path",{d:"M12 7v5l4 2"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const he=["svg",u,[["path",{d:"M2.586 17.414A2 2 0 0 0 2 18.828V21a1 1 0 0 0 1 1h3a1 1 0 0 0 1-1v-1a1 1 0 0 1 1-1h1a1 1 0 0 0 1-1v-1a1 1 0 0 1 1-1h.172a2 2 0 0 0 1.414-.586l.814-.814a6.5 6.5 0 1 0-4-4z"}],["circle",{cx:"16.5",cy:"7.5",r:".5",fill:"currentColor"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ge=["svg",u,[["rect",{width:"7",height:"9",x:"3",y:"3",rx:"1"}],["rect",{width:"7",height:"5",x:"14",y:"3",rx:"1"}],["rect",{width:"7",height:"9",x:"14",y:"12",rx:"1"}],["rect",{width:"7",height:"5",x:"3",y:"16",rx:"1"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const me=["svg",u,[["path",{d:"M21 12a9 9 0 1 1-6.219-8.56"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const fe=["svg",u,[["circle",{cx:"12",cy:"16",r:"1"}],["rect",{x:"3",y:"10",width:"18",height:"12",rx:"2"}],["path",{d:"M7 10V7a5 5 0 0 1 10 0v3"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ye=["svg",u,[["rect",{x:"16",y:"16",width:"6",height:"6",rx:"1"}],["rect",{x:"2",y:"16",width:"6",height:"6",rx:"1"}],["rect",{x:"9",y:"2",width:"6",height:"6",rx:"1"}],["path",{d:"M5 16v-3a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v3"}],["path",{d:"M12 12V8"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const be=["svg",u,[["path",{d:"M14.531 12.469 6.619 20.38a1 1 0 1 1-3-3l7.912-7.912"}],["path",{d:"M15.686 4.314A12.5 12.5 0 0 0 5.461 2.958 1 1 0 0 0 5.58 4.71a22 22 0 0 1 6.318 3.393"}],["path",{d:"M17.7 3.7a1 1 0 0 0-1.4 0l-4.6 4.6a1 1 0 0 0 0 1.4l2.6 2.6a1 1 0 0 0 1.4 0l4.6-4.6a1 1 0 0 0 0-1.4z"}],["path",{d:"M19.686 8.314a12.501 12.501 0 0 1 1.356 10.225 1 1 0 0 1-1.751-.119 22 22 0 0 0-3.393-6.319"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ve=["svg",u,[["polygon",{points:"6 3 20 12 6 21 6 3"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const we=["svg",u,[["path",{d:"M5 12h14"}],["path",{d:"M12 5v14"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Le=["svg",u,[["path",{d:"M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"}],["path",{d:"M21 3v5h-5"}],["path",{d:"M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"}],["path",{d:"M8 16H3v5"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const xe=["svg",u,[["path",{d:"M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"}],["path",{d:"M3 3v5h5"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const ke=["svg",u,[["path",{d:"M15.2 3a2 2 0 0 1 1.4.6l3.8 3.8a2 2 0 0 1 .6 1.4V19a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2z"}],["path",{d:"M17 21v-7a1 1 0 0 0-1-1H8a1 1 0 0 0-1 1v7"}],["path",{d:"M7 3v4a1 1 0 0 0 1 1h7"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Se=["svg",u,[["path",{d:"M14.536 21.686a.5.5 0 0 0 .937-.024l6.5-19a.496.496 0 0 0-.635-.635l-19 6.5a.5.5 0 0 0-.024.937l7.93 3.18a2 2 0 0 1 1.112 1.11z"}],["path",{d:"m21.854 2.147-10.94 10.939"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Ce=["svg",u,[["path",{d:"M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z"}],["path",{d:"m9 12 2 2 4-4"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Ee=["svg",u,[["rect",{width:"18",height:"18",x:"3",y:"3",rx:"2"}],["circle",{cx:"12",cy:"12",r:"1"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Me=["svg",u,[["rect",{width:"18",height:"18",x:"3",y:"3",rx:"2"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const _e=["svg",u,[["path",{d:"M3 6h18"}],["path",{d:"M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"}],["path",{d:"M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"}],["line",{x1:"10",x2:"10",y1:"11",y2:"17"}],["line",{x1:"14",x2:"14",y1:"11",y2:"17"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const $e=["svg",u,[["path",{d:"m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3"}],["path",{d:"M12 9v4"}],["path",{d:"M12 17h.01"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Ne=["svg",u,[["path",{d:"M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"}],["polyline",{points:"17 8 12 3 7 8"}],["line",{x1:"12",x2:"12",y1:"3",y2:"15"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Ae=["svg",u,[["path",{d:"M12 20h.01"}],["path",{d:"M8.5 16.429a5 5 0 0 1 7 0"}],["path",{d:"M5 12.859a10 10 0 0 1 5.17-2.69"}],["path",{d:"M19 12.859a10 10 0 0 0-2.007-1.523"}],["path",{d:"M2 8.82a15 15 0 0 1 4.177-2.643"}],["path",{d:"M22 8.82a15 15 0 0 0-11.288-3.764"}],["path",{d:"m2 2 20 20"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Pe=["svg",u,[["path",{d:"M12 20h.01"}],["path",{d:"M2 8.82a15 15 0 0 1 20 0"}],["path",{d:"M5 12.859a10 10 0 0 1 14 0"}],["path",{d:"M8.5 16.429a5 5 0 0 1 7 0"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const Re=["svg",u,[["path",{d:"M18 6 6 18"}],["path",{d:"m6 6 12 12"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const D=({icons:e={},nameAttr:r="data-lucide",attrs:t={}}={})=>{if(!Object.values(e).length)throw new Error(`Please provide an icons object.
If you want to use all the icons you can import it like:
 \`import { createIcons, icons } from 'lucide';
lucide.createIcons({icons});\``);if(typeof document>"u")throw new Error("`createIcons()` only works in a browser environment.");const o=document.querySelectorAll(`[${r}]`);if(Array.from(o).forEach(s=>O(s,{nameAttr:r,icons:e,attrs:t})),r==="data-lucide"){const s=document.querySelectorAll("[icon-name]");s.length>0&&(console.warn("[Lucide] Some icons were found with the now deprecated icon-name attribute. These will still be replaced for backwards compatibility, but will no longer be supported in v1.0 and you should switch to data-lucide"),Array.from(s).forEach(a=>O(a,{nameAttr:"icon-name",icons:e,attrs:t})))}},q={Activity:Y,ArrowDownLeft:ee,ArrowUpRight:te,CircleAlert:re,CircleCheck:ne,Clock3:oe,Copy:se,Cpu:ae,Database:ie,Download:ce,Eye:de,EyeOff:le,FileKey:ue,History:pe,KeyRound:he,LayoutDashboard:ge,LoaderCircle:me,LockKeyhole:fe,Network:ye,Pickaxe:be,Play:ve,Plus:we,RefreshCw:Le,RotateCcw:xe,Save:ke,Send:Se,ShieldCheck:Ce,Square:Me,SquareDot:Ee,Trash2:_e,TriangleAlert:$e,Upload:Ne,Wifi:Pe,WifiOff:Ae,X:Re},c={dashboard:null,history:[],ready:!1,startupCode:"starting",startupChecking:!1,dashboardRefreshing:!1,historyRefreshing:!1,recoveryPhrase:"",pendingPruneRetain:0,lastDashboardError:"",lastHistoryRefresh:0},n=e=>document.getElementById(e),Be=new TextEncoder;let U;function Te(){var e,r;return(r=(e=window.go)==null?void 0:e.main)==null?void 0:r.App}async function y(e,...r){const t=Te();if(!(t!=null&&t[e]))throw new Error(`Entropy desktop backend does not expose ${e}`);return t[e](...r)}function N(e){const r=document.createElement("i");return r.dataset.lucide=e,r}function i(e,r){const t=n(e);t&&(t.textContent=r??"--")}function T(e){return Array.isArray(e)?e:[]}function p(e,r=0){const t=Number(e);return Number.isFinite(t)?t:r}function k(e,r=12){const t=String(e||"");return t?t.length<=r?t:`${t.slice(0,r)}...`:"--"}function E(e){const r=String(e??"0"),[t,o]=r.split(".");try{const s=BigInt(t||"0").toLocaleString("en-US");return o===void 0?s:`${s}.${o}`}catch{return r}}function j(e){const r=p(e,0);if(r<=0)return"--";const t=new Date(r*1e3);return Number.isNaN(t.getTime())?"--":new Intl.DateTimeFormat("en-GB",{year:"numeric",month:"2-digit",day:"2-digit",hour:"2-digit",minute:"2-digit",second:"2-digit"}).format(t)}function De(e){let r=p(e,-1);if(r<0)return"--";if(r<1024)return`${r.toLocaleString()} B`;const t=["KiB","MiB","GiB","TiB"];let o=-1;do r/=1024,o+=1;while(r>=1024&&o<t.length-1);return`${r.toFixed(r>=100?0:r>=10?1:2)} ${t[o]}`}function m(e){return typeof e=="string"?e:e!=null&&e.message?e.message:"The operation failed"}function d(e,r="success"){clearTimeout(U);const t=n("toast");t.textContent=String(e||"Done"),t.className=`toast visible ${r}`,U=setTimeout(()=>{t.className="toast"},3600)}function g(e,r,t="Working"){if(!e)return;const o=e.querySelector("span");r?(e.dataset.originalLabel=(o==null?void 0:o.textContent)||"",e.dataset.busyLabel=t,o&&(o.textContent=t),e.classList.add("button-busy"),e.disabled=!0):(o&&e.dataset.originalLabel&&o.textContent===e.dataset.busyLabel&&(o.textContent=e.dataset.originalLabel),delete e.dataset.originalLabel,delete e.dataset.busyLabel,e.classList.remove("button-busy"),e.disabled=!1)}function F(e){return Be.encode(e).byteLength}function H(e){const r=F(e);if(r<12)throw new Error("Password must contain at least 12 UTF-8 bytes");if(r>1024)throw new Error("Password must not exceed 1024 UTF-8 bytes")}function B(e,r){const t=F(n(e).value),o=n(r);o.textContent=`${t.toLocaleString()} bytes`,o.classList.toggle("invalid",t>0&&(t<12||t>1024))}function z(e,r){return/^(?:0|[1-9]\d*)(?:\.\d{1,8})?$/.test(e)?r||!/^0(?:\.0+)?$/.test(e):!1}async function P(e,r){if(!e){d("Nothing is available to copy","error");return}try{await navigator.clipboard.writeText(e),d(r)}catch(t){d(`Clipboard access failed: ${m(t)}`,"error")}}function qe(e){const r=n("blocks-body");if(r.replaceChildren(),e.length===0){const o=r.insertRow().insertCell();o.colSpan=5,o.className="empty-cell",o.textContent="No blocks are available";return}for(const t of e){const o=r.insertRow(),s=o.insertCell(),a=document.createElement("strong");a.textContent=`#${p(t.height).toLocaleString()}`,s.append(a);const l=o.insertCell(),h=document.createElement("code");h.textContent=k(t.hash),h.title=String(t.hash||""),l.append(h),o.insertCell().textContent=j(t.timestamp),o.insertCell().textContent=p(t.transactions).toLocaleString(),o.insertCell().textContent=p(t.difficulty).toLocaleString()}}function He(e){const r=[e.online&&e.active_outbound?"Online":e.active_outbound?"Connecting":"Standby"];return p(e.height)>0&&r.push(`height #${p(e.height).toLocaleString()}`),r.push(e.bootstrap?"Bootstrap":e.discovered?e.public?"Public discovery":"Local discovery":"Manual"),e.active_outbound&&r.push("Outbound"),p(e.failures)>0&&r.push(`${p(e.failures)} failures`),r.join(" | ")}function Ve(e){const r=n("peer-list");if(r.replaceChildren(),e.length===0){const t=document.createElement("li");t.className="empty-row",t.textContent="No peers configured or discovered",r.append(t);return}for(const t of e){const o=document.createElement("li");t.last_error&&(o.title=String(t.last_error));const s=document.createElement("span");s.className=`peer-dot${t.online&&t.active_outbound?"":" offline"}`;const a=document.createElement("div");a.className="peer-main";const l=document.createElement("code");l.textContent=String(t.url||"Unknown peer");const h=document.createElement("span");h.textContent=He(t),a.append(l,h);const f=document.createElement("button");f.type="button",f.className="icon-button remove-peer",f.title=`Remove ${t.url||"peer"}`,f.setAttribute("aria-label",`Remove ${t.url||"peer"}`),f.append(N("trash-2")),f.addEventListener("click",()=>Fe(String(t.url||""),f)),o.append(s,a),t.bootstrap||o.append(f),r.append(o)}}function Oe(e){const r=n("toggle-mining");r.replaceChildren(N(e.mining?"square":"play"),document.createElement("span")),r.querySelector("span").textContent=e.mining?"Stop mining":"Start mining",r.classList.toggle("danger",!!e.mining),n("mine-once").disabled=!!e.mining,n("mining-indicator").classList.toggle("active",!!e.mining),i("mining-label",e.mining?"Mining":"Stopped")}function Ue(e){const r=p(e.height),t=p(e.best_peer_height),o=Math.max(r,t);i("best-peer-height",o.toLocaleString());const s=n("sync-progress"),a=n("sync-icon");s.classList.remove("indeterminate"),a.classList.toggle("active",!!e.syncing);const l=p(e.peer_count);e.bootstrap_enabled&&l===0?(i("sync-label","Connecting to the network"),i("sync-detail",e.bootstrap_error?"Public seeds are being retried":"Discovering public peers"),s.style.width="35%",s.classList.add("indeterminate"),a.classList.add("active")):e.syncing?(i("sync-label","Synchronizing chain"),i("sync-detail",t>r?`Local #${r.toLocaleString()} of #${t.toLocaleString()}`:"Validating peer data"),t>0?s.style.width=`${Math.min(100,r/t*100)}%`:(s.style.width="35%",s.classList.add("indeterminate"))):t>r?(i("sync-label","Peer chain is ahead"),i("sync-detail",`Local #${r.toLocaleString()} | peer #${t.toLocaleString()}`),s.style.width=`${Math.min(100,r/t*100)}%`):(i("sync-label","Chain synchronized"),i("sync-detail",`Validated through block #${r.toLocaleString()}`),s.style.width="100%"),i("network-sync",l===0?"Waiting for peers":e.syncing?`Syncing ${r.toLocaleString()} / ${o.toLocaleString()}`:"Synchronized")}function ze(e){c.dashboard=e;const r=T(e.peers),t=T(e.recent_blocks),o=p(e.height);i("confirmed-balance",E(e.confirmed_balance)),i("spendable-balance",E(e.spendable_balance)),i("wallet-address",e.address||"Unavailable"),i("wallet-page-address",e.address||"Unavailable"),i("height",o.toLocaleString()),i("peer-count",p(e.peer_count).toLocaleString()),i("pending-count",p(e.pending_count).toLocaleString()),i("difficulty",p(e.difficulty).toLocaleString()),i("target-seconds",p(e.target_block_seconds).toLocaleString()),i("listen-address",e.listen_address||"Not listening"),i("issued",E(e.issued)),i("max-supply",E(e.max_supply)),i("next-reward",E(e.next_subsidy)),i("tip-hash",k(e.tip_hash,18)),n("tip-hash").title=String(e.tip_hash||""),i("peer-badge",`${p(e.peer_count)}/${p(e.configured_peer_count)}`);const s=p(e.issued),a=p(e.max_supply),l=a>0?Math.min(100,s/a*100):0;n("supply-progress").style.width=`${Math.max(l,o>0?.35:0)}%`;const h=n("status-dot");h.classList.toggle("error",!!e.last_error),h.classList.toggle("syncing",!e.last_error&&!!e.syncing),i("node-state-label",e.last_error?"Node warning":p(e.peer_count)===0&&e.bootstrap_enabled?"Finding network":e.syncing?"Synchronizing":"Node active"),Ue(e),Oe(e),qe(t),Ve(r),i("network-protocol",e.protocol||"Unknown");const f=e.bootstrap_enabled?p(e.peer_count)>0?"Connected":e.bootstrap_ready?"Seeds loaded":e.bootstrap_error?"Retrying":"Discovering":"Disabled";i("network-bootstrap",f),n("network-bootstrap").title=String(e.bootstrap_error||""),i("network-listen",e.listen_address||"Not listening"),i("database-path",e.database_path||"Unavailable"),n("database-path").title=String(e.database_path||""),i("database-size",De(e.database_bytes));const L=n("storage-mode"),b=p(e.prune_depth),x=p(e.pruned_through);b===0&&x===0?(L.textContent="Archive",L.classList.remove("pruned")):b===0?(L.textContent="Archive going forward / previously pruned",L.classList.add("pruned")):(L.textContent=`Pruned | keep ${b.toLocaleString()}`,L.classList.add("pruned")),i("prune-depth",b>0?`${b.toLocaleString()} recent blocks`:"No future pruning"),i("pruned-through",x>0?`Block #${x.toLocaleString()}`:b>0?"Retention enabled; no eligible blocks yet":"Not pruned"),i("diagnostic-protocol",e.protocol||"Unknown"),i("diagnostic-listen",e.listen_address||"Not listening"),i("diagnostic-tip",e.tip_hash?`#${o.toLocaleString()} ${k(e.tip_hash,20)}`:"Unavailable"),n("diagnostic-tip").title=String(e.tip_hash||""),i("emission-blocks",`${p(e.emission_blocks).toLocaleString()} blocks at ${p(e.target_block_seconds).toLocaleString()} seconds`),i("last-error",e.last_error||"None"),n("last-error").classList.toggle("error-text",!!e.last_error);const w=n("health-label");w.classList.toggle("error",!!e.last_error),w.querySelector("span").textContent=e.last_error?"Warning":"Healthy",n("backup-alert").hidden=!e.wallet_needs_backup;const C=n("wallet-security-state");C.classList.toggle("warning",!!e.wallet_needs_backup),C.querySelector("span").textContent=e.wallet_needs_backup?"Backup needed":"Recovery secured",n("open-restore-backup").disabled=!!e.mining,n("open-restore-phrase").disabled=!!e.mining,n("open-restore-backup").title=e.mining?"Stop mining before restoring":"Restore encrypted backup",n("open-restore-phrase").title=e.mining?"Stop mining before restoring":"Restore recovery phrase",n("switch-archive").disabled=b===0,n("switch-archive").title=b===0?x>0?"Archive policy is active; previously deleted data remains unavailable":"Archive policy is already active":"Stop future pruning; previously deleted data will not be restored",D({icons:q})}function Ie(e){return e.coinbase?{label:"Mining reward",className:"mined",icon:"pickaxe"}:/^0(?:\.0+)?$/.test(String(e.sent??"0"))?{label:"Received transaction",className:"received",icon:"arrow-down-left"}:{label:"Sent transaction",className:"sent",icon:"arrow-up-right"}}function I(e,r){const t=document.createElement("td"),o=String(e??"0");return/^0(?:\.0+)?$/.test(o)?(t.className="zero-amount",t.textContent="--"):(t.className=r,t.textContent=`${E(o)} ENT`),t}function We(e){const r=n("history-body");if(r.replaceChildren(),i("history-count",e.length.toLocaleString()),e.length===0){const o=r.insertRow().insertCell();o.colSpan=5,o.className="empty-cell",o.textContent="No wallet transactions yet";return}for(const t of e){const o=r.insertRow();o.insertCell().textContent=j(t.timestamp);const s=o.insertCell(),a=document.createElement("div");a.className="tx-identity";const l=Ie(t),h=document.createElement("span");h.className=`tx-direction ${l.className}`,h.append(N(l.icon));const f=document.createElement("div"),L=document.createElement("strong");L.textContent=l.label;const b=document.createElement("code");b.textContent=k(t.id,22),b.title=String(t.id||""),f.append(L,b),a.append(h,f),s.append(a),o.append(I(t.received,"tx-received")),o.append(I(t.sent,"tx-sent"));const x=document.createElement("td"),w=document.createElement("span");if(t.pending)w.className="status-badge pending",w.append(N("clock-3"),document.createTextNode("Pending"));else{const C=p(t.confirmations);w.className="status-badge confirmed",w.append(N("circle-check"),document.createTextNode(`${C.toLocaleString()} confirmation${C===1?"":"s"}`)),t.block_height!=null&&(w.title=`Block #${p(t.block_height).toLocaleString()}`)}x.append(w),o.append(x)}D({icons:q})}async function v(){if(!(!c.ready||c.dashboardRefreshing)){c.dashboardRefreshing=!0;try{const e=await y("GetDashboard");ze(e),c.lastDashboardError=""}catch(e){const r=m(e);n("status-dot").classList.add("error"),i("node-state-label","Node offline"),r!==c.lastDashboardError&&d(r,"error"),c.lastDashboardError=r}finally{c.dashboardRefreshing=!1}}}async function S(e=!1){if(!c.ready||c.historyRefreshing)return;c.historyRefreshing=!0;const r=n("refresh-history");e&&(r.disabled=!0,r.classList.add("button-busy"));try{const t=T(await y("GetTransactionHistory",100));c.history=t,c.lastHistoryRefresh=Date.now(),We(t),i("history-updated",`Updated ${new Intl.DateTimeFormat("en-GB",{hour:"2-digit",minute:"2-digit",second:"2-digit"}).format(new Date)}`)}catch(t){if((e||c.history.length===0)&&d(m(t),"error"),c.history.length===0){const o=n("history-body");o.replaceChildren();const a=o.insertRow().insertCell();a.colSpan=5,a.className="empty-cell",a.textContent="Transaction history is unavailable"}}finally{c.historyRefreshing=!1,e&&(r.disabled=!1,r.classList.remove("button-busy"))}}function M(e){n("startup-loading").hidden=e!=="loading",n("migration-form").hidden=e!=="migration",n("startup-failed").hidden=e!=="failed"}async function R(){if(!(c.startupChecking||c.ready)){c.startupChecking=!0;try{const e=await y("GetStartupState");if(c.startupCode=e.code||"starting",i("startup-message",e.message||"Opening wallet and ledger..."),e.ready||e.code==="ready"){c.ready=!0,n("startup-overlay").classList.remove("visible"),await Promise.all([v(),S()]);return}n("startup-overlay").classList.add("visible"),e.code==="wallet_migration_required"?M("migration"):e.code==="startup_failed"?(i("startup-error",e.message||"Unknown startup error"),M("failed")):M("loading")}catch(e){c.startupCode="startup_failed",i("startup-error",m(e)),M("failed"),n("startup-overlay").classList.add("visible")}finally{c.startupChecking=!1}}}function je(e){for(const r of document.querySelectorAll("[data-view]"))r.classList.toggle("active",r.dataset.view===e);for(const r of document.querySelectorAll("[data-view-panel]")){const t=r.dataset.viewPanel===e;r.hidden=!t,r.classList.toggle("active",t)}e==="transactions"&&S(!0)}function G(e){for(const r of e.querySelectorAll('input[type="password"], textarea'))r.value="";for(const r of e.querySelectorAll('input[type="checkbox"]'))r.checked=!1;e.id==="recovery-dialog"&&(c.recoveryPhrase="",n("recovery-grid").replaceChildren(),n("recovery-placeholder").hidden=!1,n("recovery-content").hidden=!0,n("confirm-recovery").disabled=!0),e.id==="prune-dialog"&&(c.pendingPruneRetain=0,n("confirm-prune").disabled=!0),A()}function _(e){const r=n(e);G(r),r.open||r.showModal()}function $(e){const r=n(e);r.open&&r.close()}function A(){B("migration-password","migration-password-bytes"),B("export-password","export-password-bytes"),B("restore-backup-password","restore-password-bytes");const e=n("restore-phrase").value.trim().split(/\s+/).filter(Boolean).length;i("restore-word-count",`${e} / 24 words`)}async function Fe(e,r){if(e){g(r,!0,"Removing");try{const t=await y("RemovePeer",e);d(t.message||"Peer removed"),await v()}catch(t){d(m(t),"error"),g(r,!1)}}}document.querySelectorAll("[data-view]").forEach(e=>{e.addEventListener("click",()=>je(e.dataset.view))});document.querySelectorAll("[data-close-dialog]").forEach(e=>{e.addEventListener("click",()=>$(e.dataset.closeDialog))});document.querySelectorAll("dialog").forEach(e=>{e.addEventListener("click",r=>{r.target===e&&e.close()}),e.addEventListener("close",()=>G(e))});n("copy-address").addEventListener("click",()=>{var e;return P((e=c.dashboard)==null?void 0:e.address,"Address copied")});n("copy-wallet-page-address").addEventListener("click",()=>{var e;return P((e=c.dashboard)==null?void 0:e.address,"Address copied")});n("copy-database-path").addEventListener("click",()=>{var e;return P((e=c.dashboard)==null?void 0:e.database_path,"Database path copied")});n("send-form").addEventListener("submit",async e=>{e.preventDefault();const t=e.currentTarget.querySelector('button[type="submit"]'),o=n("send-to").value.trim(),s=n("send-amount").value.trim(),a=n("send-fee").value.trim();if(!o){d("Recipient address is required","error");return}if(!z(s,!1)){d("Amount must be positive with no more than 8 decimal places","error");return}if(!z(a,!0)){d("Fee must be zero or positive with no more than 8 decimal places","error");return}g(t,!0,"Broadcasting");try{const l=await y("SendTransaction",o,s,a);d(`${l.message||"Transaction submitted"}: ${k(l.id)}`),n("send-amount").value="",await Promise.all([v(),S()])}catch(l){d(m(l),"error")}finally{g(t,!1)}});n("toggle-mining").addEventListener("click",async()=>{if(!c.dashboard)return;const e=n("toggle-mining"),r=!c.dashboard.mining;g(e,!0,r?"Starting":"Stopping");try{const t=await y("SetMining",r);d(t.message||(r?"Mining started":"Mining stopping")),await v()}catch(t){d(m(t),"error")}finally{g(e,!1)}});n("mine-once").addEventListener("click",async()=>{const e=n("mine-once");g(e,!0,"Mining block");try{const r=await y("MineOneBlock");d(`${r.message||"Block mined"}: ${k(r.id)}`),await Promise.all([v(),S()])}catch(r){d(m(r),"error")}finally{g(e,!1)}});n("peer-form").addEventListener("submit",async e=>{e.preventDefault();const r=e.currentTarget,t=n("peer-url"),o=r.querySelector('button[type="submit"]'),s=t.value.trim();if(s){o.disabled=!0;try{const a=await y("AddPeer",s);d(a.message||"Peer added"),t.value="",await v()}catch(a){d(m(a),"error")}finally{o.disabled=!1}}});n("refresh-history").addEventListener("click",()=>S(!0));n("backup-alert-action").addEventListener("click",()=>_("recovery-dialog"));n("open-recovery").addEventListener("click",()=>_("recovery-dialog"));n("open-export").addEventListener("click",()=>_("export-dialog"));n("open-restore-backup").addEventListener("click",()=>_("restore-backup-dialog"));n("open-restore-phrase").addEventListener("click",()=>_("restore-phrase-dialog"));n("prune-form").addEventListener("submit",e=>{var l,h;e.preventDefault();const r=n("prune-retain").value.trim();if(!/^[1-9]\d*$/.test(r)){d("Retention must be a whole number","error");return}const t=Number(r);if(!Number.isSafeInteger(t)||t<120||t>31536e3){d("Retention must be between 120 and 31,536,000 blocks","error");return}const o=p((l=c.dashboard)==null?void 0:l.height),s=p((h=c.dashboard)==null?void 0:h.pruned_through),a=Math.max(0,o-t);c.pendingPruneRetain=t,s>a?i("prune-impact",`The ledger is already pruned through block #${s.toLocaleString()}. Deleted data will not be restored; future pruning will retain the newest ${t.toLocaleString()} blocks.`):a===0?i("prune-impact",`Current height is #${o.toLocaleString()}. No existing body is eligible yet; future pruning will retain the newest ${t.toLocaleString()} blocks.`):i("prune-impact",`Block and transaction bodies plus undo records through block #${a.toLocaleString()} will be permanently removed. The newest ${t.toLocaleString()} blocks remain complete.`),_("prune-dialog"),c.pendingPruneRetain=t});n("prune-confirm-check").addEventListener("change",e=>{n("confirm-prune").disabled=!e.currentTarget.checked});n("switch-archive").addEventListener("click",async()=>{var r;const e=n("switch-archive");g(e,!0,"Switching");try{const t=await y("PruneLedger",0);d(`${t.message||"Archive policy enabled"}. Previously pruned data remains unavailable.`,"info"),await v()}catch(t){d(m(t),"error")}finally{g(e,!1),p((r=c.dashboard)==null?void 0:r.prune_depth)===0&&(e.disabled=!0)}});n("confirm-prune").addEventListener("click",async()=>{const e=n("confirm-prune"),r=c.pendingPruneRetain;if(!(!r||!n("prune-confirm-check").checked)){g(e,!0,"Pruning");try{const t=await y("PruneLedger",r);d(t.message||"Ledger pruning completed"),$("prune-dialog"),await v()}catch(t){d(m(t),"error"),g(e,!1)}}});n("reveal-recovery").addEventListener("click",async()=>{const e=n("reveal-recovery");g(e,!0,"Decrypting");try{const t=String(await y("GetRecoveryPhrase")).trim().split(/\s+/).filter(Boolean);if(t.length!==24)throw new Error("Backend returned an invalid recovery phrase");c.recoveryPhrase=t.join(" ");const o=n("recovery-grid");o.replaceChildren(),t.forEach((s,a)=>{const l=document.createElement("li"),h=document.createElement("span");h.textContent=String(a+1);const f=document.createElement("code");f.textContent=s,l.append(h,f),o.append(l)}),n("recovery-placeholder").hidden=!0,n("recovery-content").hidden=!1}catch(r){const t=m(r);i("recovery-action-detail",t),d(t,"error")}finally{g(e,!1)}});n("copy-recovery").addEventListener("click",()=>P(c.recoveryPhrase,"Recovery phrase copied"));n("recovery-confirm-check").addEventListener("change",e=>{n("confirm-recovery").disabled=!e.currentTarget.checked});n("confirm-recovery").addEventListener("click",async()=>{const e=n("confirm-recovery");if(!(!c.recoveryPhrase||!n("recovery-confirm-check").checked)){g(e,!0,"Saving");try{const r=await y("ConfirmWalletRecovery");d(r.message||"Wallet recovery confirmed"),$("recovery-dialog"),await v()}catch(r){d(m(r),"error"),g(e,!1)}}});n("export-form").addEventListener("submit",async e=>{var a;e.preventDefault();const t=e.currentTarget.querySelector('button[type="submit"]'),o=n("export-password").value,s=n("export-confirm").value;try{if(H(o),o!==s)throw new Error("Password confirmation does not match")}catch(l){d(m(l),"error");return}g(t,!0,"Encrypting");try{const l=await y("ExportWalletBackup",o);d(l.message||"Wallet backup exported",(a=l.message)!=null&&a.toLowerCase().includes("cancel")?"info":"success"),$("export-dialog"),await v()}catch(l){d(m(l),"error")}finally{n("export-password").value="",n("export-confirm").value="",g(t,!1)}});n("restore-backup-form").addEventListener("submit",async e=>{var s;e.preventDefault();const t=e.currentTarget.querySelector('button[type="submit"]'),o=n("restore-backup-password").value;try{if(H(o),!n("restore-backup-check").checked)throw new Error("Confirm active wallet replacement")}catch(a){d(m(a),"error");return}g(t,!0,"Restoring");try{const a=await y("RestoreWalletBackup",o);d(a.id?`${a.message}: ${k(a.id,18)}`:a.message,(s=a.message)!=null&&s.toLowerCase().includes("cancel")?"info":"success"),$("restore-backup-dialog"),c.history=[],await Promise.all([v(),S()])}catch(a){d(m(a),"error")}finally{n("restore-backup-password").value="",g(t,!1)}});n("restore-phrase-form").addEventListener("submit",async e=>{e.preventDefault();const t=e.currentTarget.querySelector('button[type="submit"]'),o=n("restore-phrase").value.trim().split(/\s+/).filter(Boolean);if(o.length!==24){d("Recovery phrase must contain exactly 24 words","error");return}if(!n("restore-phrase-check").checked){d("Confirm active wallet replacement","error");return}const s=o.join(" ");g(t,!0,"Restoring");try{const a=await y("RestoreWalletMnemonic",s);d(`${a.message||"Wallet restored"}: ${k(a.id,18)}`),$("restore-phrase-dialog"),c.history=[],await Promise.all([v(),S()])}catch(a){d(m(a),"error")}finally{n("restore-phrase").value="",g(t,!1)}});n("migration-form").addEventListener("submit",async e=>{var a;e.preventDefault();const t=e.currentTarget.querySelector('button[type="submit"]'),o=n("migration-password").value,s=n("migration-confirm").value;try{if(H(o),o!==s)throw new Error("Password confirmation does not match")}catch(l){d(m(l),"error");return}g(t,!0,"Encrypting");try{const l=await y("MigrateLegacyWallet",o),h=(a=l.message)==null?void 0:a.toLowerCase().includes("cancel");d(l.message||"Legacy wallet migrated",h?"info":"success"),h||(c.startupCode="starting",M("loading"),await R())}catch(l){d(m(l),"error")}finally{n("migration-password").value="",n("migration-confirm").value="",A(),g(t,!1)}});n("retry-startup").addEventListener("click",async()=>{c.startupCode="starting",M("loading");try{await y("RetryStartup")}catch(e){d(m(e),"error")}await R()});for(const e of[n("migration-password"),n("export-password"),n("restore-backup-password")])e.addEventListener("input",A);n("restore-phrase").addEventListener("input",A);window.addEventListener("beforeunload",()=>{document.querySelectorAll('input[type="password"], textarea').forEach(e=>{e.value=""}),c.recoveryPhrase=""});async function Ge(){if(!c.ready){c.startupCode==="starting"&&await R();return}await v(),Date.now()-c.lastHistoryRefresh>=5e3&&await S()}D({icons:q});A();R();setInterval(Ge,1200);
