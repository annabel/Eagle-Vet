const menuButton = document.querySelector('.menu-toggle');
const navigation = document.querySelector('#navigation');
const triggers = [...document.querySelectorAll('.nav-trigger')];
function closeDropdowns(except) {
 for (const trigger of triggers) if(trigger !== except) { trigger.setAttribute('aria-expanded','false'); document.getElementById(trigger.getAttribute('aria-controls')).hidden = true; }
}
for (const trigger of triggers) trigger.addEventListener('click', () => {
 const open=trigger.getAttribute('aria-expanded')!=='true'; closeDropdowns(trigger);
 trigger.setAttribute('aria-expanded',String(open));document.getElementById(trigger.getAttribute('aria-controls')).hidden=!open;
});
menuButton?.addEventListener('click',()=>{const open=menuButton.getAttribute('aria-expanded')!=='true';menuButton.setAttribute('aria-expanded',String(open));navigation.classList.toggle('is-open',open);if(!open)closeDropdowns();});
document.addEventListener('click',e=>{if(!e.target.closest('.nav-group'))closeDropdowns();});
document.addEventListener('keydown',e=>{if(e.key==='Escape'){const open=triggers.find(t=>t.getAttribute('aria-expanded')==='true');closeDropdowns();if(open){open.focus();}else if(menuButton?.getAttribute('aria-expanded')==='true'){menuButton.setAttribute('aria-expanded','false');navigation.classList.remove('is-open');menuButton.focus();}}});
navigation?.addEventListener('click',e=>{if(e.target.closest('a')){closeDropdowns();navigation.classList.remove('is-open');menuButton.setAttribute('aria-expanded','false');}});
// Demo forms intentionally have no action, field names, storage, or network requests.
for (const form of document.querySelectorAll('[data-demo-form]')) {
 const submit = form.querySelector('[data-demo-submit]');
 submit.disabled = false;
 form.addEventListener('submit',e=>e.preventDefault());
 submit.addEventListener('click',()=>{
  if(!form.reportValidity())return;
  const message=form.querySelector('[role="status"]');message.hidden=false;
  message.textContent='This is a design preview. Nothing was sent or saved. Please call 208-939-8285 to contact the clinic.';
  message.focus();
 });
}
for(const group of document.querySelectorAll('[data-required-group]')) {
 const checkboxes=[...group.querySelectorAll('input[type="checkbox"]')];
 const validate=()=>checkboxes[0].setCustomValidity(checkboxes.some(c=>c.checked)?'':'Please choose at least one option.');
 group.addEventListener('change',validate);validate();
}

// Load the same IDEXX Neo integration used by the current clinic site, on request.
const portalButton=document.querySelector('[data-open-portal]');
let portalInstance;
let portalScript;
portalButton?.addEventListener('click',async()=>{
 const status=document.querySelector('.portal-status');
 portalButton.disabled=true;status.hidden=false;status.textContent='Opening the secure portal…';
 try{
  if(!window.BeeFree){
   portalScript??=new Promise((resolve,reject)=>{
    const css=document.createElement('link');css.rel='stylesheet';css.href='https://us.idexxneo.com/public/clientportal/v1/clientportal.css';document.head.append(css);
    const script=document.createElement('script');script.src='https://us.idexxneo.com/public/clientportal/v1/clientportal.js';script.onload=resolve;script.onerror=()=>reject(new Error('Portal unavailable'));document.head.append(script);
   });
   await portalScript;
  }
  portalInstance??=new window.BeeFree.ClientPortal({server:'https://us.idexxneo.com/',companyId:'8597',branch:'1'});
  portalInstance.showClientPortal();status.textContent='The secure portal has opened.';
 }catch{
  portalScript=undefined;status.textContent='The portal could not open. Please try again or contact the clinic at 208-939-8285.';
 }finally{portalButton.disabled=false;}
});

// Native dialogs keep keyboard focus inside the biography and support Escape.
let biographyOpener;
for (const button of document.querySelectorAll('[data-bio-open]')) {
 button.addEventListener('click',()=>{
  const dialog=document.getElementById(button.dataset.bioOpen);
  if(!dialog || dialog.open)return;
  biographyOpener=button;
  dialog.showModal();
  document.body.classList.add('bio-open');
 });
}
for (const dialog of document.querySelectorAll('.bio-dialog')) {
 dialog.querySelector('[data-bio-close]').addEventListener('click',()=>dialog.close());
 dialog.addEventListener('click',event=>{
  if(event.target!==dialog)return;
  const rect=dialog.getBoundingClientRect();
  if(event.clientX<rect.left || event.clientX>rect.right || event.clientY<rect.top || event.clientY>rect.bottom)dialog.close();
 });
 dialog.addEventListener('close',()=>{
  document.body.classList.remove('bio-open');
  biographyOpener?.focus({preventScroll:true});
 });
}
