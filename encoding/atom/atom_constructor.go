package atom

/*
// from ADE, example of how to construct a new AtomContainer.:
static void FINF_RegisterService(ForwarderInformationInfoPtr theModuleInfo)
{
	CXD_AtomContainerPtr		msgAC = NULL;
	UINT32				rc = 0;

	CXD_Atom_CreateContainer(&msgAC);
	CXD_Atom_SetType(msgAC, CONTAINER_IS_PARENT, EVENT_SERVICE_REGISTER);

	CXD_AtomPath_SetUI32(msgAC, EVENT_SERVICE_REGISTER_VERSION, 1);
	CXD_AtomPath_SetFC32(msgAC, EVENT_SERVICE_REGISTER_SERVICEID,
			FORWARDER_SERVICE_ID);
	CXD_AtomPath_SetUI32(msgAC, EVENT_SERVICE_REGISTER_SERVICEVERSION
			, FORWARDER_SERVICE_VERSION);
	CXD_AtomPath_SetFC32(msgAC, EVENT_SERVICE_REGISTER_SERVICESCOPE,
			FORWARDER_SERVICE_SCOPE);
	CXD_AtomPath_SetCSTR(msgAC, EVENT_SERVICE_REGISTER_SERVICENAME,
			FORWARDER_SERVICE_NAME);
	CXD_AtomPath_SetFC32(msgAC, EVENT_SERVICE_REGISTER_SERVICESTATE,
			SERVICESTATE_ENABLED);
	CXD_AtomPath_SetUI32(msgAC, EVENT_SERVICE_REGISTER_SERVICEPROCESSID,
			ADE_Process_GetPID());

	rc = ADE_Message_PostContainer(theModuleInfo->ServicePID,
			ADE_Process_GetNID(), &msgAC);
	REQUIRES(rc == noErr);

	theModuleInfo->ServiceRegistered = true;

}

// Here is an example of a message, should make it easy and succinct to
// construct this sort of thing on the fly:
       SADD:CONT:
            MVER:UI32:1
            TSID:UI64:1
            SVID:FC32:'DING'
            SVVS:UI32:1
            SVPS:CONT:
                  SVNM:CSTR:"DICOM Ingest Service"
                  SVST:FC32:'ENBL'
                  SVPI:UI32:24
                  LHND:UI32:1
                  FWDC:UI32:1
                  TCPI:UI32:1
                  SIPA:IP32:192.168.170.93
                  SPRT:UI32:5104
            END
       END
*/

/*
Example of bundle attribute structure
GODS:CONT:
    BVER:UI32:1
    BTIM:UI64:1
    GOPT:CONT:
#        "Option"
        AVER:UI32:2
        ATIM:UI64:1
        AVTP:FC32:'CSTR'
        APER:FC32:'READ'
        AVAL:CONT:
            0x00000000:UI32:1
        END
    END
    GOVL:CONT:
#        "Value"
        AVER:UI32:2
        ATIM:UI64:1
        AVTP:FC32:'CSTR'
        APER:FC32:'READ'
        AVAL:CONT:
            0x00000000:UI32:1
        END
    END
END
*/

// Attempt to design a go type that could be automatically converted
// into an Atom object
type attributeRows map[uint32]interface{}
type avalContainer struct {
	index uint32 `atom: 0x00000000:UI32`
	attributeRows
}
type AttributeBranches []AttributeBranch

// FIXME how to succintly define AVTP?
type AttributeBranch struct {
	AVER uint32
	ATIM uint64 `atom: CTIM:UI32` // eg. allowing name and/or type to be explicitly defined instead of assumed from go type and var name
	AVTP string `atom: AVTP:CSTR`
	APER string `atom: APER:CSTR` // otherwise default to CSTR i suppose
	AVAL avalContainer
}

// FIXME: can I actually get struct field names? It must be possible because Fprintf("%+v", myStructInstance) does it

func NewAttributeBranch(name string, avtp string, aper string) *AttributeBranch {
	ab := AttributeBranch{AVER: 0, ATIM: 0, AVTP: avtp, APER: aper}
	return &ab
}

func NewAttributeContainer(name string, containers ...AttributeBranch) *AttributeContainer {
	abs := AttributeBranches(containers)
	return &AttributeContainer{name: name, AttributeBranches: abs}
}
func (a *AttributeContainer) AddRow(values ...interface{}) {
}

// Confirm that this struct conforms to attribute container structure
func (a *AttributeContainer) validate() error {
	var err error
	return err
}
func (a *AttributeContainer) EncodeBinary() {
}
func (a *AttributeContainer) EncodeText() {
}

// FIXME: where to put the Atom object?
type AttributeContainer struct {
	name string
	BVER uint32
	BTIM uint64
	AttributeBranches
}

// substitute for main, just to experiment with this syntax
func attrScratchSpace() {
	gods := NewAttributeContainer("GODS",
		*NewAttributeBranch("GOPT", "CSTR", "READ"),
		*NewAttributeBranch("GOVL", "CSTR", "READ"),
	)
	gods.AddRow("MyDog", "HasFleas")

	//gods.AVER = 2
	//gods.GOPT.AVTP = "UI32"
	//gods.GOPT.AVTP = "UI32"
}
