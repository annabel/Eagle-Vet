const icons = {
 heart: '<path d="M24 40C14 32 5 24 5 16 5 5 19 3 24 12 29 3 43 5 43 16 43 24 34 32 24 40Z"/><path d="m12 24 7 0 4-9 5 17 4-8h5"/>',
 cross: '<path d="M17 6h14v12h12v14H31v12H17V32H5V18h12Z"/>',
 tooth: '<path d="M24 9C11 1 4 9 8 23c3 9 2 20 8 20 4 0 3-13 8-13s4 13 8 13c6 0 5-11 8-20C44 9 37 1 24 9Z"/><path d="M24 9l7 4"/>',
 rabbit: '<path d="M16 26C7 5 12-4 18 8l7 17C23 5 29-3 32 8l-1 20c13 5 13 17-2 17H14C1 45 3 30 16 26Z"/><path d="M30 34h.1M38 39l-5 1"/>'
};
const menu = document.querySelector('.menu-toggle');
const nav = document.querySelector('#navigation');
menu.addEventListener('click', () => {
 const expanded = menu.getAttribute('aria-expanded') !== 'true';
 menu.setAttribute('aria-expanded', String(expanded));
 nav.classList.toggle('is-open', expanded);
});
nav.addEventListener('click', event => {if(event.target.closest('a')) closeMenu();});
function closeMenu(){menu.setAttribute('aria-expanded','false');nav.classList.remove('is-open');}
document.addEventListener('keydown',event=>{if(event.key==='Escape'){closeMenu();}});
async function loadServices(){
 try {
 const response=await fetch('content.json'); if(!response.ok)throw new Error('Content unavailable');
 const {services}=await response.json();
 const container=document.querySelector('#services');
 for(const service of services){
 const link=document.createElement('a');link.className='service';link.href=`https://www.eaglevethosp.com/services#${encodeURIComponent(service.anchor)}`;
 const icon=document.createElement('span');icon.className=`service-icon ${service.color}`;icon.innerHTML=`<svg viewBox="0 0 48 50" aria-hidden="true">${icons[service.icon]||icons.heart}</svg>`;
 const title=document.createElement('h3');title.textContent=service.title;
 const description=document.createElement('p');description.textContent=service.description;
 const arrow=document.createElement('span');arrow.className='service-arrow';arrow.setAttribute('aria-hidden','true');arrow.textContent='↗';
 link.append(icon,title,description,arrow);container.append(link);
 }
 }catch(error){document.querySelector('#services').innerHTML='<p>Explore our full range of care on the <a href="https://www.eaglevethosp.com/services">services page</a>.</p>';console.error(error);}
}
loadServices();
document.querySelector('#year').textContent=new Date().getFullYear();
