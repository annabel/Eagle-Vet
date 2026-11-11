import http from 'node:http';
import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
const root=fileURLToPath(new URL('.',import.meta.url));
const mime={'.html':'text/html; charset=utf-8','.css':'text/css; charset=utf-8','.js':'text/javascript; charset=utf-8','.json':'application/json','.png':'image/png','.jpg':'image/jpeg','.woff2':'font/woff2'};
const server=http.createServer(async(req,res)=>{
 try{
 const pathname=decodeURIComponent(new URL(req.url,'http://localhost').pathname);
 const file=path.resolve(root,'.'+(pathname==='/'?'/index.html':pathname));
 if(!file.startsWith(root)||file.includes('/.')){res.writeHead(403).end('Forbidden');return;}
 const body=await readFile(file);res.writeHead(200,{'Content-Type':mime[path.extname(file)]||'application/octet-stream','Cache-Control':'no-store'});res.end(body);
 }catch{res.writeHead(404).end('Not found');}
});
const preferredPort = Number(process.env.PORT ?? 4173);
const maxAttempts = 10;
let attempts = 0;
let port = preferredPort;

if (!Number.isInteger(port) || port < 0 || port > 65535) {
 console.error('PORT must be a whole number between 0 and 65535.');
 process.exit(1);
}

server.on('error', error => {
 if (error.code === 'EADDRINUSE' && attempts < maxAttempts - 1 && port < 65535) {
  console.log(`Port ${port} is in use; trying ${port + 1}…`);
  port += 1;
  attempts += 1;
  server.listen(port, '127.0.0.1');
  return;
 }
 console.error(`Could not start the preview: ${error.message}`);
 process.exitCode = 1;
});

server.on('listening', () => {
 console.log(`Eagle Vet demo: http://127.0.0.1:${server.address().port}`);
});

server.listen(port, '127.0.0.1');
