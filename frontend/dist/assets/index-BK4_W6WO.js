(function(){const s=document.createElement("link").relList;if(s&&s.supports&&s.supports("modulepreload"))return;for(const o of document.querySelectorAll('link[rel="modulepreload"]'))r(o);new MutationObserver(o=>{for(const i of o)if(i.type==="childList")for(const l of i.addedNodes)l.tagName==="LINK"&&l.rel==="modulepreload"&&r(l)}).observe(document,{childList:!0,subtree:!0});function t(o){const i={};return o.integrity&&(i.integrity=o.integrity),o.referrerPolicy&&(i.referrerPolicy=o.referrerPolicy),o.crossOrigin==="use-credentials"?i.credentials="include":o.crossOrigin==="anonymous"?i.credentials="omit":i.credentials="same-origin",i}function r(o){if(o.ep)return;o.ep=!0;const i=t(o);fetch(o.href,i)}})();/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const v=(e,s,t=[])=>{const r=document.createElementNS("http://www.w3.org/2000/svg",e);return Object.keys(s).forEach(o=>{r.setAttribute(o,String(s[o]))}),t.length&&t.forEach(o=>{const i=v(...o);r.appendChild(i)}),r};var S=([e,s,t])=>v(e,s,t);/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const I=e=>Array.from(e.attributes).reduce((s,t)=>(s[t.name]=t.value,s),{}),$=e=>typeof e=="string"?e:!e||!e.class?"":e.class&&typeof e.class=="string"?e.class.split(" "):e.class&&Array.isArray(e.class)?e.class:"",T=e=>e.flatMap($).map(t=>t.trim()).filter(Boolean).filter((t,r,o)=>o.indexOf(t)===r).join(" "),k=e=>e.replace(/(\w)(\w*)(_|-|\s*)/g,(s,t,r)=>t.toUpperCase()+r.toLowerCase()),w=(e,{nameAttr:s,icons:t,attrs:r})=>{var b;const o=e.getAttribute(s);if(o==null)return;const i=k(o),l=t[i];if(!l)return console.warn(`${e.outerHTML} icon name was not found in the provided icons object.`);const f=I(e),[M,N,A]=l,h={...N,"data-lucide":o,...r,...f},y=T(["lucide",`lucide-${o}`,f,r]);y&&Object.assign(h,{class:y});const L=S([M,h,A]);return(b=e.parentNode)==null?void 0:b.replaceChild(L,e)};/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const a={xmlns:"http://www.w3.org/2000/svg",width:24,height:24,viewBox:"0 0 24 24",fill:"none",stroke:"currentColor","stroke-width":2,"stroke-linecap":"round","stroke-linejoin":"round"};/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const B=["svg",a,[["path",{d:"M7 7h10v10"}],["path",{d:"M7 17 17 7"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const P=["svg",a,[["rect",{width:"14",height:"14",x:"8",y:"8",rx:"2",ry:"2"}],["path",{d:"M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const O=["svg",a,[["rect",{width:"16",height:"16",x:"4",y:"4",rx:"2"}],["rect",{width:"6",height:"6",x:"9",y:"9",rx:"1"}],["path",{d:"M15 2v2"}],["path",{d:"M15 20v2"}],["path",{d:"M2 15h2"}],["path",{d:"M2 9h2"}],["path",{d:"M20 15h2"}],["path",{d:"M20 9h2"}],["path",{d:"M9 2v2"}],["path",{d:"M9 20v2"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const j=["svg",a,[["path",{d:"M14.531 12.469 6.619 20.38a1 1 0 1 1-3-3l7.912-7.912"}],["path",{d:"M15.686 4.314A12.5 12.5 0 0 0 5.461 2.958 1 1 0 0 0 5.58 4.71a22 22 0 0 1 6.318 3.393"}],["path",{d:"M17.7 3.7a1 1 0 0 0-1.4 0l-4.6 4.6a1 1 0 0 0 0 1.4l2.6 2.6a1 1 0 0 0 1.4 0l4.6-4.6a1 1 0 0 0 0-1.4z"}],["path",{d:"M19.686 8.314a12.501 12.501 0 0 1 1.356 10.225 1 1 0 0 1-1.751-.119 22 22 0 0 0-3.393-6.319"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const D=["svg",a,[["polygon",{points:"6 3 20 12 6 21 6 3"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const q=["svg",a,[["path",{d:"M5 12h14"}],["path",{d:"M12 5v14"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const H=["svg",a,[["path",{d:"M14.536 21.686a.5.5 0 0 0 .937-.024l6.5-19a.496.496 0 0 0-.635-.635l-19 6.5a.5.5 0 0 0-.024.937l7.93 3.18a2 2 0 0 1 1.112 1.11z"}],["path",{d:"m21.854 2.147-10.94 10.939"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const z=["svg",a,[["rect",{width:"18",height:"18",x:"3",y:"3",rx:"2"}],["circle",{cx:"12",cy:"12",r:"1"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const F=["svg",a,[["rect",{width:"18",height:"18",x:"3",y:"3",rx:"2"}]]];/**
 * @license lucide v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const E=({icons:e={},nameAttr:s="data-lucide",attrs:t={}}={})=>{if(!Object.values(e).length)throw new Error(`Please provide an icons object.
If you want to use all the icons you can import it like:
 \`import { createIcons, icons } from 'lucide';
lucide.createIcons({icons});\``);if(typeof document>"u")throw new Error("`createIcons()` only works in a browser environment.");const r=document.querySelectorAll(`[${s}]`);if(Array.from(r).forEach(o=>w(o,{nameAttr:s,icons:e,attrs:t})),s==="data-lucide"){const o=document.querySelectorAll("[icon-name]");o.length>0&&(console.warn("[Lucide] Some icons were found with the now deprecated icon-name attribute. These will still be replaced for backwards compatibility, but will no longer be supported in v1.0 and you should switch to data-lucide"),Array.from(o).forEach(i=>w(i,{nameAttr:"icon-name",icons:e,attrs:t})))}},_={ArrowUpRight:B,Copy:P,Cpu:O,Pickaxe:j,Play:D,Plus:q,Send:H,Square:F,SquareDot:z},C={name:"Entropy",symbol:"ENT",address:"ent1c4f1a78d3420f6eb6cc89f29bd54bc144b0ea8d4a77cd41",confirmed_balance:"0.12683918",spendable_balance:"0.12683918",height:2,tip_hash:"000002afcab83521ec7111b5432e21779ad33ce9c7cdf863b7fa5285fe183a28",difficulty:22,pending_count:1,peer_count:1,configured_peer_count:2,peers:[{url:"http://192.168.1.20:47821",online:!0},{url:"http://10.0.0.8:47821",online:!1}],mining:!1,listen_address:"[::]:47821",issued:"0.12683918",max_supply:"2000000",target_block_seconds:10,emission_blocks:31536e3,next_subsidy:"0.06341959",last_error:"",recent_blocks:[{height:2,hash:"000002afcab83521ec7111b5432e21779ad33ce9c7cdf863b7fa5285fe183a28",timestamp:1783923020,transactions:2,difficulty:22},{height:1,hash:"00000cb251e9a90c2269228f97adb4040fec174ebe2aa5f3215eb2f5a25aa800",timestamp:1783923010,transactions:1,difficulty:22},{height:0,hash:"c7108201a36db97765911f4362c4af3f24294e5031e17d52f1115f7b7712e435",timestamp:1783900800,transactions:0,difficulty:0}]},n=Object.fromEntries(["confirmed-balance","spendable-balance","wallet-address","height","peer-count","pending-count","difficulty","target-seconds","listen-address","issued","max-supply","supply-progress","tip-hash","blocks-body","peer-list","peer-badge","mining-label","mining-indicator","toggle-mining","mine-once","status-dot","node-state-label","toast","next-reward"].map(e=>[e,document.getElementById(e)]));let g=C,x;function G(){var e,s;return(s=(e=window.go)==null?void 0:e.main)==null?void 0:s.App}async function u(e,...s){const t=G();return t!=null&&t[e]?t[e](...s):e==="GetDashboard"?C:(await new Promise(r=>setTimeout(r,350)),{message:"Preview action completed"})}function p(e,s=12){return!e||e.length<=s?e||"--":`${e.slice(0,s)}...`}function R(e){return new Intl.DateTimeFormat("zh-CN",{hour:"2-digit",minute:"2-digit",second:"2-digit",month:"2-digit",day:"2-digit"}).format(new Date(e*1e3))}function U(e){g=e,n["confirmed-balance"].textContent=e.confirmed_balance,n["spendable-balance"].textContent=e.spendable_balance,n["wallet-address"].textContent=e.address,n.height.textContent=e.height.toLocaleString(),n["peer-count"].textContent=e.peer_count,n["pending-count"].textContent=e.pending_count,n.difficulty.textContent=e.difficulty,n["target-seconds"].textContent=e.target_block_seconds,n["listen-address"].textContent=e.listen_address||"Starting",n.issued.textContent=e.issued,n["max-supply"].textContent=Number(e.max_supply).toLocaleString(),n["next-reward"].textContent=e.next_subsidy,n["tip-hash"].textContent=p(e.tip_hash,18),n["tip-hash"].title=e.tip_hash,n["peer-badge"].textContent=`${e.peer_count}/${e.configured_peer_count}`,n["status-dot"].classList.toggle("error",!!e.last_error),n["node-state-label"].textContent=e.last_error?"Node warning":"Node active";const s=Math.min(100,Number(e.issued)/Number(e.max_supply)*100);if(n["supply-progress"].style.width=`${Math.max(s,e.height>0?.35:0)}%`,n["blocks-body"].innerHTML=e.recent_blocks.map(t=>`
    <tr>
      <td><strong>#${t.height.toLocaleString()}</strong></td>
      <td><code title="${t.hash}">${p(t.hash)}</code></td>
      <td>${R(t.timestamp)}</td>
      <td>${t.transactions}</td>
      <td>${t.difficulty}</td>
    </tr>
  `).join(""),n["peer-list"].replaceChildren(),e.peers.length===0){const t=document.createElement("li");t.className="empty-row",t.textContent="No peers connected",n["peer-list"].append(t)}else for(const t of e.peers){const r=document.createElement("li"),o=document.createElement("span"),i=document.createElement("code");o.className=`peer-dot${t.online?"":" offline"}`,i.textContent=t.url,r.append(o,i),n["peer-list"].append(r)}n["mining-label"].textContent=e.mining?"Mining":"Stopped",n["mining-indicator"].classList.toggle("active",e.mining),n["toggle-mining"].classList.toggle("danger",e.mining),n["toggle-mining"].innerHTML=e.mining?'<i data-lucide="square"></i><span>Stop mining</span>':'<i data-lucide="play"></i><span>Start mining</span>',n["mine-once"].disabled=e.mining,E({icons:_})}function c(e,s="success"){clearTimeout(x),n.toast.textContent=e,n.toast.className=`toast visible ${s}`,x=setTimeout(()=>{n.toast.className="toast"},3200)}function m(e){return typeof e=="string"?e:(e==null?void 0:e.message)||"Action failed"}async function d(){try{U(await u("GetDashboard"))}catch(e){n["status-dot"].classList.add("error"),n["node-state-label"].textContent="Node offline",c(m(e),"error")}}document.getElementById("copy-address").addEventListener("click",async()=>{await navigator.clipboard.writeText(g.address),c("Address copied")});document.getElementById("send-form").addEventListener("submit",async e=>{e.preventDefault();const s=e.currentTarget.querySelector("button[type=submit]");s.disabled=!0;try{const t=await u("SendTransaction",document.getElementById("send-to").value.trim(),document.getElementById("send-amount").value.trim(),document.getElementById("send-fee").value.trim());c(`${t.message}: ${p(t.id)}`),document.getElementById("send-amount").value="",await d()}catch(t){c(m(t),"error")}finally{s.disabled=!1}});n["toggle-mining"].addEventListener("click",async()=>{n["toggle-mining"].disabled=!0;try{const e=await u("SetMining",!g.mining);c(e.message),await d()}catch(e){c(m(e),"error")}finally{n["toggle-mining"].disabled=!1}});n["mine-once"].addEventListener("click",async()=>{n["mine-once"].disabled=!0;try{const e=await u("MineOneBlock");c(e.message),await d()}catch(e){c(m(e),"error")}finally{n["mine-once"].disabled=!1}});document.getElementById("peer-form").addEventListener("submit",async e=>{e.preventDefault();const s=document.getElementById("peer-url");try{const t=await u("AddPeer",s.value.trim());c(t.message),s.value="",await d()}catch(t){c(m(t),"error")}});E({icons:_});d();setInterval(d,1e3);
