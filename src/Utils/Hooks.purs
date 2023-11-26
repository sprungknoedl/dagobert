module Dagobert.Utils.Hooks where

import Prelude

import Effect (Effect)
import Effect.Class (class MonadEffect, liftEffect)

updateTo :: forall a m r.
  MonadEffect m 
  => { push :: a -> Effect Unit | r }
  -> a 
  -> m Unit
updateTo state newState = liftEffect $ state.push newState

infixr 0 updateTo as <~