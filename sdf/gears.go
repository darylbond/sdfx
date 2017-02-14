//-----------------------------------------------------------------------------
/*

Involute Gears

*/
//-----------------------------------------------------------------------------

package sdf

import "math"

//-----------------------------------------------------------------------------

// return the involute coordinate
// r = base radius
// theta = involute angle
func involute(r, theta float64) V2 {
	c := math.Cos(theta)
	s := math.Sin(theta)
	return V2{
		r * (c + theta*s),
		r * (s - theta*c),
	}
}

// return the involute angle
// r = base radius
// d = involute radial distance
func involute_angle(r, d float64) float64 {
	x := d / r
	return math.Sqrt(x*x - 1)
}

//-----------------------------------------------------------------------------

// Generate an SDF2 polygon for a single involute tooth
// number_teeth = number of gear teeth
// gear_module = pitch circle diameter / number of gear teeth
// root_radius = radius at tooth root
// base_radius = radius at the base of the involute
// outer_radius = radius at the outside of the tooth
// backlash = backlash expressed as units of pitch circumference
// facets = number of facets for involute flank
func InvoluteGearTooth(
	number_teeth int,
	gear_module float64,
	root_radius float64,
	base_radius float64,
	outer_radius float64,
	backlash float64,
	facets int,
) SDF2 {

	pitch_radius := float64(number_teeth) * gear_module / 2.0

	// work out the angular extent of the tooth on the base radius
	pitch_point := involute(base_radius, involute_angle(base_radius, pitch_radius))
	face_angle := math.Atan2(pitch_point.Y, pitch_point.X)
	backlash_angle := backlash / (2.0 * pitch_radius)
	center_angle := PI/(2.0*float64(number_teeth)) + face_angle - backlash_angle

	// work out the angles over which the involute will be used
	start_angle := involute_angle(base_radius, Max(base_radius, root_radius))
	stop_angle := involute_angle(base_radius, outer_radius)
	dtheta := (stop_angle - start_angle) / float64(facets)

	v := make([]V2, 2*(facets+1)+1)

	// lower tooth face
	m := Rotate(-center_angle)
	angle := start_angle
	for i := 0; i <= facets; i++ {
		v[i] = m.MulPosition(involute(base_radius, angle))
		angle += dtheta
	}

	// upper tooth face (mirror the lower point)
	for i := 0; i <= facets; i++ {
		p := v[facets-i]
		v[facets+1+i] = V2{p.X, -p.Y}
	}

	// add the origin to make the polygon a tooth wedge
	v[2*(facets+1)] = V2{0, 0}

	return NewPolySDF2(v)
}

//-----------------------------------------------------------------------------

// Generate an SDF2 polygon for an involute gear
// number_teeth = number of gear teeth
// gear_module = pitch circle diameter / number of gear teeth
// pressure_angle = gear pressure angle (radians)
// backlash = backlash expressed as units of pitch circumference
// clearance = additional root clearance
// ring_width = width of ring wall (from root circle)
// facets = number of facets for involute flank
func InvoluteGear(
	number_teeth int,
	gear_module float64,
	pressure_angle float64,
	backlash float64,
	clearance float64,
	ring_width float64,
	facets int,
) SDF2 {

	// pitch radius
	pitch_radius := float64(number_teeth) * gear_module / 2.0

	// base circle radius
	base_radius := pitch_radius * math.Cos(pressure_angle)

	// addendum: radial distance from pitch circle to outside circle
	addendum := gear_module * 1.0
	// dedendum: radial distance from pitch circle to root circle
	dedendum := addendum + clearance

	outer_radius := pitch_radius + addendum
	root_radius := pitch_radius - dedendum
	ring_radius := root_radius - ring_width

	tooth := InvoluteGearTooth(
		number_teeth,
		gear_module,
		root_radius,
		base_radius,
		outer_radius,
		backlash,
		facets,
	)

	gear := NewRotateSDF2(tooth, number_teeth, Rotate2d(TAU/float64(number_teeth)))
	root := NewCircleSDF2(root_radius)
	ring := NewCircleSDF2(ring_radius)

	return NewDifferenceSDF2(NewUnionSDF2(gear, root), ring)
}

//-----------------------------------------------------------------------------
// 2D Gear Rack

type GearRack struct {
	tooth  SDF2    // polygon for rack tooth
	pitch  float64 // tooth to tooth pitch
	length float64 // half the total rack length
	bb     Box2    // bounding box
}

// Create a 2D gear rack.
// number_teeth = number of rack teeth
// gear_module = pitch circle diameter / number of gear teeth
// pressure_angle = gear pressure angle (radians)
// backlash = backlash expressed as units of pitch circumference
// base_height = height of rack base
func NewGearRack(
	number_teeth float64,
	gear_module float64,
	pressure_angle float64,
	backlash float64,
	base_height float64,
) SDF2 {
	s := GearRack{}

	// addendum: distance from pitch line to top of tooth
	addendum := gear_module * 1.0
	// dedendum: distance from pitch line to root of tooth
	dedendum := gear_module * 1.25
	// total tooth height
	tooth_height := base_height + addendum + dedendum
	// tooth_pitch: tooth to tooth distance along pitch line
	pitch := gear_module * PI

	// x size of tooth flank
	dx := (addendum + dedendum) * math.Tan(pressure_angle)
	// 1/2 x size of tooth top
	dxt := ((pitch / 2.0) - dx) / 2.0
	// x size of backlash
	bl := backlash / 2.0

	// create a half tooth profile centered on the y-axis
	tooth := []V2{
		V2{pitch, 0},
		V2{pitch, base_height},
		V2{dx + dxt - bl, base_height},
		V2{dxt - bl, tooth_height},
		V2{-pitch, tooth_height},
		V2{-pitch, 0},
	}

	s.tooth = NewPolySDF2(tooth)
	s.pitch = pitch
	s.length = pitch * number_teeth / 2.0
	s.bb = Box2{V2{-s.length, 0}, V2{s.length, tooth_height}}
	return &s
}

// Return the minimum distance to the gear rack.
func (s *GearRack) Evaluate(p V2) float64 {
	// map p.X back to the [0,half_pitch) domain
	p0 := V2{Abs(SawTooth(p.X, s.pitch)), p.Y}
	// get the tooth profile distance
	d0 := s.tooth.Evaluate(p0)
	// create a region for the rack length
	d1 := Abs(p.X) - s.length
	// return the intersection
	return Max(d0, d1)
}

// Return the bounding box for the gear rack.
func (s *GearRack) BoundingBox() Box2 {
	return s.bb
}

//-----------------------------------------------------------------------------
