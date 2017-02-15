//-----------------------------------------------------------------------------
/*

Cams

*/
//-----------------------------------------------------------------------------

package sdf

import (
	"fmt"
	"math"
)

//-----------------------------------------------------------------------------
// Cam Type 1: Flat Flank Cam.

type Cam1 struct {
	distance    float64 // center to center circle distance
	base_radius float64 // radius of base circle
	nose_radius float64 // radius of nose circle
	a           V2      // lower point on flank line
	u           V2      // normalised line vector for flank
	l           float64 // length of flank line
	bb          Box2    // bounding box
}

// Create a 2D cam profile.
// The profile is made from 2 circles and straight line flanks.
// The base circle is centered on the origin.
// The nose circle is located on the positive y axis.
// distance = circle to circle center distance
// base_radius = radius of base circle
// nose_radius = radius of nose circle
func NewCam1(distance, base_radius, nose_radius float64) SDF2 {
	s := Cam1{}
	s.distance = distance
	s.base_radius = base_radius
	s.nose_radius = nose_radius
	// work out the flank line
	sin := (base_radius - nose_radius) / distance
	cos := math.Sqrt(1 - sin*sin)
	// first point on line
	s.a = V2{cos, sin}.MulScalar(base_radius)
	// second point on line
	b := V2{cos, sin}.MulScalar(nose_radius).Add(V2{0, distance})
	// line information
	u := b.Sub(s.a)
	s.u = u.Normalize()
	s.l = u.Length()
	// work out the bounding box
	s.bb = Box2{V2{-base_radius, -base_radius}, V2{base_radius, distance + nose_radius}}
	return &s
}

// Return the minimum distance to the cam.
func (s *Cam1) Evaluate(p V2) float64 {
	// we have symmetry about the y-axis
	p0 := V2{Abs(p.X), p.Y}
	// vector to first point of flank line
	v := p0.Sub(s.a)
	// work out the t-parameter of the projection onto the flank line
	t := v.Dot(s.u)
	var d float64
	if t < 0 {
		// the nearest point is on the major circle
		d = p0.Length() - s.base_radius
	} else if t <= s.l {
		// the nearest point is on the flank line
		d = v.Dot(V2{s.u.Y, -s.u.X})
	} else {
		// the nearest point is on the minor circle
		d = p0.Sub(V2{0, s.distance}).Length() - s.nose_radius
	}
	return d
}

// Return the bounding box for the cam.
func (s *Cam1) BoundingBox() Box2 {
	return s.bb
}

//-----------------------------------------------------------------------------
// Cam Type 2: Three Arc Cam.

type Cam2 struct {
	distance     float64 // center to center circle distance
	base_radius  float64 // radius of base circle
	nose_radius  float64 // radius of nose circle
	flank_radius float64 // radius of flank circle
	flank_center V2      // center of flank circle (+ve x-axis flank arc)
	theta_base   float64 // base/flank intersection angle wrt flank center
	theta_nose   float64 // nose/flank intersection angle wrt flank center
	bb           Box2    // bounding box
}

// Create a 2D cam profile.
// The profile is made from 2 circles and circular flank arcs.
// The base circle is centered on the origin.
// The nose circle is located on the positive y axis.
// The flank arcs are tangential to the base and nose circles.
// distance = circle to circle center distance
// base_radius = radius of major circle
// nose_radius = radius of minor circle
// flank_radius = radius of flank arc
func NewCam2(distance, base_radius, nose_radius, flank_radius float64) SDF2 {
	s := Cam2{}
	s.distance = distance
	s.base_radius = base_radius
	s.nose_radius = nose_radius
	s.flank_radius = flank_radius
	// work out the center for the flank radius
	// the flank arc center must lie on the intersection
	// of two circles about the base/nose circles
	r0 := flank_radius - base_radius
	r1 := flank_radius - nose_radius
	y := ((r0 * r0) - (r1 * r1) + (distance * distance)) / (2.0 * distance)
	x := -math.Sqrt((r0 * r0) - (y * y)) // < 0 result, +ve x-axis flank arc
	s.flank_center = V2{x, y}
	// work out theta for the intersection of flank arc and base radius
	p := V2{0, 0}.Sub(s.flank_center).Normalize().MulScalar(flank_radius)
	p = p.Sub(s.flank_center)
	s.theta_base = math.Atan2(p.Y, p.X)
	// work out theta for the intersection of flank arc and nose radius
	p = V2{0, distance}.Sub(s.flank_center).Normalize().MulScalar(flank_radius)
	p = p.Sub(s.flank_center)
	s.theta_nose = math.Atan2(p.Y, p.X)
	// work out the bounding box
	s.bb = Box2{V2{-base_radius, -base_radius}, V2{base_radius, distance + nose_radius}}
	return &s
}

// Return the minimum distance to the cam.
func (s *Cam2) Evaluate(p V2) float64 {
	// we have symmetry about the y-axis
	p0 := V2{Abs(p.X), p.Y}
	// work out the theta angle wrt the flank center
	v := p0.Sub(s.flank_center)
	t := math.Atan2(v.Y, v.X)
	// work out the minimum distance
	var d float64
	if t < s.theta_base {
		// the closest point is on the base radius
		d = p0.Length() - s.base_radius
	} else if t > s.theta_nose {
		// the closest point is on the nose radius
		d = p0.Sub(V2{0, s.distance}).Length() - s.nose_radius
	} else {
		// the closest point is on the flank radius
		d = v.Length() - s.flank_radius
	}
	return d
}

// Return the bounding box for the cam.
func (s *Cam2) BoundingBox() Box2 {
	return s.bb
}

//-----------------------------------------------------------------------------

// Create a cam profile from design parameters.
// cam_type = flat_flank, three_arc
// lift = follower lift distance from base circle
// duration = angle over which the follower lifts from the base circle
// max_diameter = maximum diameter of cam rotation
func MakeCam(cam_type string, lift, duration, max_diameter float64) (SDF2, error) {

	// check the arguments
	if max_diameter <= 0 {
		return nil, fmt.Errorf("max_diameter <= 0")
	}
	if lift <= 0 {
		return nil, fmt.Errorf("lift <= 0")
	}
	if duration <= 0 || duration >= PI {
		return nil, fmt.Errorf("invalid duration")
	}

	base_radius := (max_diameter / 2.0) - lift
	if base_radius <= 0 {
		return nil, fmt.Errorf("base_radius <= 0")
	}

	delta := duration / 2.0

	if cam_type == "flat_flank" {
		c := math.Cos(delta)
		nose_radius := base_radius - (lift*c)/(1-c)
		if nose_radius <= 0 {
			return nil, fmt.Errorf("nose_radius <= 0")
		}
		distance := base_radius + lift - nose_radius
		return NewCam1(distance, base_radius, nose_radius), nil
	} else if cam_type == "three_arc" {
		// TODO

	} else {
		return nil, fmt.Errorf("unknown cam_type")
	}

	return nil, nil
}

//-----------------------------------------------------------------------------
