import {test} from 'node:test'
import assert from 'node:assert/strict'
import {mkdtempSync,mkdirSync,copyFileSync,readFileSync,writeFileSync,rmSync} from 'node:fs'
import {tmpdir} from 'node:os'
import {join} from 'node:path'
import {spawnSync} from 'node:child_process'

for(const scenario of ['idempotent','version drift','source drift'])test(scenario,()=>{
 const root=mkdtempSync(join(tmpdir(),'workbraid-routing-patch-'))
 try{
  mkdirSync(join(root,'scripts'));mkdirSync(join(root,'node_modules/cytoscape/dist'),{recursive:true})
  copyFileSync(new URL('patch-cytoscape.mjs',import.meta.url),join(root,'scripts/patch-cytoscape.mjs'))
  for(const file of ['package.json','dist/cytoscape.esm.mjs','dist/cytoscape.cjs.js'])copyFileSync(new URL('../node_modules/cytoscape/'+file,import.meta.url),join(root,'node_modules/cytoscape',file))
  if(scenario==='version drift')writeFileSync(join(root,'node_modules/cytoscape/package.json'),JSON.stringify({version:'3.34.2'}))
  if(scenario==='source drift'){const file=join(root,'node_modules/cytoscape/dist/cytoscape.esm.mjs');writeFileSync(file,readFileSync(file,'utf8')+'\n// unexpected source\n')}
  const run=()=>spawnSync(process.execPath,[join(root,'scripts/patch-cytoscape.mjs')],{encoding:'utf8'})
  const result=run()
  if(scenario==='idempotent'){assert.equal(result.status,0,result.stderr);assert.equal(run().status,0)}else{assert.notEqual(result.status,0);assert.match(result.stderr,/requires Cytoscape 3.34.1|unexpected Cytoscape source/)}
 }finally{rmSync(root,{recursive:true,force:true})}
})
