// Phase 3.3: preserve Cytoscape's intersection midpoint and clipping, correcting
// only the directed normal for unbundled curves. No runtime renderer hook.
import {readFileSync, writeFileSync} from 'node:fs'
import {createHash} from 'node:crypto'

const root = new URL('../node_modules/cytoscape/', import.meta.url)
if (JSON.parse(readFileSync(new URL('package.json',root),'utf8')).version !== '3.34.1') throw Error('WorkBraid routing patch requires Cytoscape 3.34.1')
const original = `      vectorNormInverse = _this$findMidptPtsEtc2.vectorNormInverse;
    var adjustedMidpt = {`
const patched = `      vectorNormInverse = _this$findMidptPtsEtc2.vectorNormInverse;
    // WorkBraid: directed center normal; retain the exact intersection midpoint.
    if (edgeIsUnbundled) {
      var routeSource = edge.source().position();
      var routeTarget = edge.target().position();
      var routeDX = routeTarget.x - routeSource.x;
      var routeDY = routeTarget.y - routeSource.y;
      var routeLength = Math.hypot(routeDX, routeDY);
      if (routeLength > 0 && Number.isFinite(routeLength)) {
        vectorNormInverse = { x: -routeDY / routeLength, y: routeDX / routeLength };
      }
    }
    var adjustedMidpt = {`
const expected = {
 'dist/cytoscape.esm.mjs':'57f306c96a2197421ec438370599278013420f2f03bd29e3b6483b41b157951e',
 'dist/cytoscape.cjs.js':'3f5bd9a99bafff60baf7e03d72dc280ea6d353c307c293aecca4544e010be0c0',
}
// Validate both consumed entry points before changing either. Also verify the
// complete original bytes on repeated builds, not only the replacement anchor.
const changes = Object.entries(expected).map(([path, hash]) => {
 const file = new URL(path,root), current = readFileSync(file,'utf8')
 const source = current.includes(patched) ? current.replace(patched,original) : current
 if (createHash('sha256').update(source).digest('hex') !== hash || source.split(original).length !== 2) throw Error(`WorkBraid routing patch: unexpected Cytoscape source in ${path}`)
 return {file,current,next:source.replace(original,patched)}
})
for (const {file,current,next} of changes) if(current !== next) writeFileSync(file,next)
console.log('Verified Cytoscape 3.34.1 directed routing patch (ESM and CommonJS)')
