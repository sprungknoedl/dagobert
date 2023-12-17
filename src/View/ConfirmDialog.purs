module Dagobert.View.ConfirmDialog where

import Prelude

import Dagobert.Utils.HTML (dangerButton, modal, secondaryButton)
import Dagobert.Utils.Icons (xMark)
import Deku.Core (Nut, fixed)
import Deku.DOM as D
import Deku.DOM.Attributes as DA
import Deku.DOM.Listeners as DL
import Effect (Effect)

confirmDialog :: forall a row.
  { accept :: a -> Effect Unit
  , reject  :: Effect Unit
  | row }
  -> a
  -> Nut
confirmDialog { accept, reject } input =
  modal $ fixed
    [ D.header [ DA.klass_ "p-6 px-8 border-b border-b-slate-700 flex justify-between" ]
      [ D.h4  [ DA.klass_ "font-bold text-slate-200" ] [ D.text_ "Confirm" ]
      , D.div_ 
          [ D.button [ DA.xtype_ "button", DL.runOn DL.click $ pure reject ]
          [ xMark $ DA.klass_ "w-6 h-6" ] 
          ]
      ]
      
    , D.section [ DA.klass_ "p-8 flex flex-col gap-6" ] [
      D.text_ "Are you sure you want to delete this entry? This action is permanent and can not be undone."
    ]

    , D.footer [ DA.klass_ "p-6 px-8 border-t border-t-slate-700 flex gap-4" ] 
      [ dangerButton [ DA.xtype_ "submit", DL.runOn DL.click $ pure (accept input) ] [ D.text_ "Confirm" ]
      , secondaryButton [ DA.xtype_ "button", DL.runOn DL.click $ pure reject ] [ D.text_ "Cancel" ]
      ]
    ] 

