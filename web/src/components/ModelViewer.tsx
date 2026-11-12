import { useEffect, useRef, useState } from 'react'
import * as THREE from 'three'
import { STLLoader } from 'three/examples/jsm/loaders/STLLoader.js'
import { ThreeMFLoader } from 'three/examples/jsm/loaders/3MFLoader.js'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'

import type { Asset } from '../api/types'
import { assetUrl } from '../lib/urls'

/**
 * Podgląd modelu 3D (.stl, .3mf) w przeglądarce.
 *
 * Ten moduł jest ładowany leniwie (React.lazy w ItemDetail) — three.js waży
 * kilkaset kilobajtów i nie ma powodu, żeby pobierała go osoba oglądająca
 * notatkę albo zdjęcie.
 */
export default function ModelViewer({ asset }: { asset: Asset }) {
  const mountRef = useRef<HTMLDivElement>(null)
  const [state, setState] = useState<'loading' | 'ready' | 'error'>('loading')
  const [message, setMessage] = useState('')

  useEffect(() => {
    const mount = mountRef.current
    if (!mount) return

    // `disposed` chroni przed operowaniem na scenie po odmontowaniu
    // komponentu — wczytywanie modelu jest asynchroniczne i może skończyć się
    // już po tym, jak użytkownik przejdzie na inną stronę.
    let disposed = false
    let frame = 0

    const scene = new THREE.Scene()
    const camera = new THREE.PerspectiveCamera(45, 1, 0.1, 5000)

    const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true })
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
    mount.appendChild(renderer.domElement)

    const controls = new OrbitControls(camera, renderer.domElement)
    controls.enableDamping = true
    controls.dampingFactor = 0.08

    // Światło: jedno kierunkowe z góry-przodu plus miękkie wypełnienie.
    // Bez wypełnienia spody modelu są czarne i nie widać kształtu.
    scene.add(new THREE.HemisphereLight(0xffffff, 0x33302c, 2.2))
    const key = new THREE.DirectionalLight(0xffffff, 1.8)
    key.position.set(1, 1.4, 1)
    scene.add(key)

    function resize() {
      const { clientWidth, clientHeight } = mount!
      if (clientWidth === 0 || clientHeight === 0) return
      renderer.setSize(clientWidth, clientHeight, false)
      camera.aspect = clientWidth / clientHeight
      camera.updateProjectionMatrix()
    }

    const observer = new ResizeObserver(resize)
    observer.observe(mount)
    resize()

    /** Ustawia kamerę tak, żeby model wypełnił kadr, niezależnie od skali pliku. */
    function frameObject(object: THREE.Object3D) {
      const box = new THREE.Box3().setFromObject(object)
      const size = box.getSize(new THREE.Vector3())
      const center = box.getCenter(new THREE.Vector3())

      // Model przesuwamy do środka układu, zamiast przesuwać kamerę —
      // dzięki temu orbitowanie kręci się wokół obiektu, a nie obok niego.
      object.position.sub(center)

      const radius = Math.max(size.x, size.y, size.z) / 2 || 1
      const distance = radius / Math.sin((camera.fov * Math.PI) / 360)

      camera.position.set(distance * 0.7, distance * 0.5, distance * 0.9)
      camera.near = distance / 100
      camera.far = distance * 100
      camera.updateProjectionMatrix()

      controls.target.set(0, 0, 0)
      controls.update()
    }

    const material = new THREE.MeshStandardMaterial({
      color: 0xb08a52, // ciepły bursztyn — ten sam charakter co akcent w UI
      roughness: 0.55,
      metalness: 0.05,
      flatShading: false,
    })

    const url = assetUrl(asset.id, 'original')
    const isThreeMF = asset.filename.toLowerCase().endsWith('.3mf')

    function onLoaded(object: THREE.Object3D) {
      if (disposed) return
      scene.add(object)
      frameObject(object)
      setState('ready')
    }

    function onFailed(err: unknown) {
      if (disposed) return
      setMessage(err instanceof Error ? err.message : 'nie udało się wczytać modelu')
      setState('error')
    }

    if (isThreeMF) {
      new ThreeMFLoader().load(
        url,
        (group) => {
          group.traverse((child) => {
            if (child instanceof THREE.Mesh) child.material = material
          })
          onLoaded(group)
        },
        undefined,
        onFailed,
      )
    } else {
      new STLLoader().load(
        url,
        (geometry) => {
          // Pliki STL nie niosą normalnych wierzchołków w formie, której
          // oczekuje cieniowanie gładkie — trzeba je policzyć samemu.
          geometry.computeVertexNormals()
          onLoaded(new THREE.Mesh(geometry, material))
        },
        undefined,
        onFailed,
      )
    }

    function animate() {
      frame = requestAnimationFrame(animate)
      controls.update()
      renderer.render(scene, camera)
    }
    animate()

    return () => {
      disposed = true
      cancelAnimationFrame(frame)
      observer.disconnect()
      controls.dispose()

      // WebGL nie zwalnia pamięci karty graficznej sam z siebie —
      // bez tego każde wejście na stronę modelu zostawiałoby po sobie bufory.
      scene.traverse((child) => {
        if (child instanceof THREE.Mesh) {
          child.geometry.dispose()
          if (Array.isArray(child.material)) child.material.forEach((m) => m.dispose())
          else child.material.dispose()
        }
      })
      renderer.dispose()
      renderer.domElement.remove()
    }
  }, [asset.id, asset.filename])

  return (
    <div className="relative aspect-4/3 w-full overflow-hidden rounded-md bg-surface">
      <div ref={mountRef} className="size-full" />

      {state === 'loading' && (
        <p className="absolute inset-0 flex items-center justify-center font-mono text-xs text-faint">
          wczytuję model…
        </p>
      )}
      {state === 'error' && (
        <p className="absolute inset-0 flex items-center justify-center px-6 text-center text-sm text-danger">
          Nie udało się wczytać modelu. {message}
        </p>
      )}
      {state === 'ready' && (
        <p className="pointer-events-none absolute bottom-3 left-4 font-mono text-2xs text-faint">
          przeciągnij, żeby obrócić · kółko przybliża
        </p>
      )}
    </div>
  )
}
