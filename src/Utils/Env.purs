module Dagobert.Utils.Env where

import Prelude

import Dagobert.Data.Case (Case)
import Dagobert.Route (Route)
import Data.Maybe (Maybe)
import Effect (Effect)
import FRP.Poll (Poll)

type Env =
    { kase  :: Poll (Maybe Case)
    , setKase :: Maybe Case -> Effect Unit
    , route :: Poll (Route)
    , setRoute :: Route -> Effect Unit
    }