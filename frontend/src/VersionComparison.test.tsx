import {afterEach,expect,it,vi} from 'vitest'
import {cleanup,render,screen,waitFor} from '@testing-library/react'
import {VersionComparison} from './VersionComparison'

afterEach(()=>{cleanup();vi.restoreAllMocks()})
function mockVersions(){
 return vi.spyOn(globalThis,'fetch').mockImplementation(async(url,init)=>{
  if(url==='/api/projects')return new Response(JSON.stringify({projects:[{store_id:'store',slug:'project',name:'Project'}]}))
  const input=JSON.parse(String(init?.body))
  return new Response(JSON.stringify({ok:true,result:{accepted_revision:'current',versions:input.source==='accepted'?[{selector:{kind:'accepted',revision:'current'},label:'Current Accepted',revision:'current'}]:[]}}))
 })
}
it('pins current Accepted once on bare entry and leaves After empty',async()=>{
 window.history.replaceState({},'','/projects/project/compare')
 mockVersions();render(<VersionComparison/> )
 await waitFor(()=>expect(new URLSearchParams(window.location.search).get('before')).toBe(JSON.stringify({kind:'accepted',revision:'current'})))
 expect(new URLSearchParams(window.location.search).has('after')).toBe(false)
 await waitFor(()=>expect(screen.getByLabelText('Version',{selector:'#Before-version'})).toHaveValue(JSON.stringify({kind:'accepted',revision:'current'})))
 expect(screen.getByRole('button',{name:'Report'})).toBeDisabled()
})
it.each([
 '?store_id=store',
 '?before='+encodeURIComponent(JSON.stringify({kind:'accepted',revision:'old'})),
 '?after='+encodeURIComponent(JSON.stringify({kind:'proposal',change_set_id:'gone',state:'stale',side:'candidate'})),
 '?before=null',
 '?before=invalid',
])('preserves explicit partial, stale or invalid entry %s',async query=>{
 window.history.replaceState({},'','/projects/project/compare'+query)
 const fetch=mockVersions();render(<VersionComparison/> )
 await screen.findByRole('heading',{name:query.includes('invalid')?'Comparison unavailable':'Compare versions'})
 await waitFor(()=>expect(fetch).toHaveBeenCalled())
 expect(window.location.search).toBe(query)
 expect(fetch.mock.calls.some(([,init])=>init?.body&&JSON.parse(String(init.body)).limit===1)).toBe(false)
})
