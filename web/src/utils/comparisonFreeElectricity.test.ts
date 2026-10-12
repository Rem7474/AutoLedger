import { describe, expect, it } from 'vitest'
import { applyComparisonDefaults, comparisonStepErrorKey, emptyComparisonForm } from './comparisonForm'
describe('free electricity in projections',()=>{
 it('accepts explicit zero but not missing, negative or nonfinite prices',()=>{
  const f=emptyComparisonForm(false);f.ice.purchase_price=20000;f.tracked.purchase_price=25000
  for(const v of [0,'0']){f.tracked.eur_per_kwh=v as any;expect(comparisonStepErrorKey(2,f,false)).toBe('')}
  for(const v of ['',null,undefined,-1,NaN,Infinity]){f.tracked.eur_per_kwh=v as any;expect(comparisonStepErrorKey(2,f,false)).toBe('comparison.errors.electricityPrice')}
 })
 it('preserves an explicitly free default',()=>{const f=emptyComparisonForm(false);applyComparisonDefaults(f,{annual_km:12000,maintenance_yearly:0,insurance_yearly:0,tracked_eur_per_kwh:0});expect(f.tracked.eur_per_kwh).toBe(0)})
})
