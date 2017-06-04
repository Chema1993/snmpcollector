import { FormBuilder, Validators, FormArray, FormGroup, FormControl} from '@angular/forms';
import { ValidationService } from './validation.service'

export class AvailableTableActions {

  //AvailableOptions result depeding on component type
  public availableOptions : Array<any>;

  // type can be : device,...
  // data is the passed extraData when declaring AvailableTableActions on each component
  checkComponentType(type, data?) : any {
    switch (type) {
      case 'device':
        return this.getDeviceAvailableActions(data);
      default:
        return null;
      }
  }

  constructor (componentType : string, extraData : any) {
    this.availableOptions = this.checkComponentType(componentType, extraData);
  }

  //Devices Available Acions:
  getDeviceAvailableActions (data) : any {
    let tableAvailableActions = [
    //Remove Action
      {'title': 'Remove', 'content' :
        {'type' : 'button','action' : 'RemoveAllSelected'}
      },
    //Change Property Action
      {'title': 'Change property', 'content' :
        {'type' : 'selector', 'action' : 'ChangeProperty', 'options' : [
          {'title' : 'Active', 'type':'boolean', 'options' : [
            'true','false'
            ]
          },
          {'title' : 'LogLevel', 'type':'boolean', 'options' : [
            'panic','fatal','error','warning','info','debug'
            ]
          },
          {'title' : 'ConcurrentGather', 'type':'boolean', 'options' : [
            'true','false'
            ]
          },
          {'title' : 'DisableBulk', 'type':'boolean', 'options' : [
            'true','false'
            ]
          },
          {'title' : 'SnmpDebug', 'type':'boolean', 'options' : [
            'true','false'
            ]
          },
          {'title': 'Freq','type':'input',' options':
            new FormGroup({
              formControl : new FormControl('', Validators.compose([Validators.required, ValidationService.uintegerNotZeroValidator]))
            })
          },
          {'title': 'MaxRepetitions','type':'input', 'options':
            new FormGroup({
              formControl : new FormControl('', Validators.compose([Validators.required,ValidationService.uinteger8NotZeroValidator]))
            })
          },
          {'title' : 'MeasurementGroups', 'type':'multiselector', 'options' :
            data[0]
          },
          {'title' : 'MeasFilters', 'type':'multiselector', 'options' :
            data[1]
          },
          {'title' : 'ExtraTags', 'type':'input', 'options' :
            new FormGroup({
              formControl : new FormControl('', Validators.compose([ValidationService.noWhiteSpaces, ValidationService.extraTags]))
            })
          }
        ]},
      },
      //AppendProperty
      {'title': 'AppendProperty', 'content' :
        {'type' : 'selector', 'action' : 'AppendProperty', 'options' : [
          {'title' : 'MeasurementGroups', 'type':'multiselector', 'options' :
            data[0]
          },
          {'title' : 'MeasFilters', 'type':'multiselector', 'options' :
            data[1]
          },
          {'title' : 'ExtraTags', 'type':'input', 'options':
            new FormGroup({
              formControl : new FormControl('', Validators.compose([ValidationService.noWhiteSpaces, ValidationService.extraTags]))
            })
          }
          ]
        }
      }
    ];
    return tableAvailableActions;
  }
}
