module Dagobert.Utils.Hooks where

import Prelude

import Effect (Effect)
import Effect.Class (class MonadEffect, liftEffect)
import FRP.Poll (Poll)

type PollIO a = { poll ∷ Poll a, push ∷ a -> Effect Unit }

updateTo :: forall a m r.
  MonadEffect m 
  => { push :: a -> Effect Unit | r }
  -> a 
  -> m Unit
updateTo state newState = liftEffect $ state.push newState

infixr 0 updateTo as <~